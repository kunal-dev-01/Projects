package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"shared/logging" 

	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	URL        string
	Host       string
	VHost      string
	MaxRetries int
	RetryDelay time.Duration
}

type RabbitBroker struct {
	mu     sync.RWMutex
	conn   *amqp.Connection
	cfg    Config
	closed bool
	ctx    context.Context
	cancel context.CancelFunc
}

// New ab bina kisi extra logger parameter ke chalega, cleaner API!
func New(cfg Config) *RabbitBroker {
	ctx, cancel := context.WithCancel(context.Background())
	return &RabbitBroker{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (b *RabbitBroker) Connect(ctx context.Context) error {
	if err := b.connectWithRetry(ctx); err != nil {
		return err
	}
	go b.watchConnection()
	return nil
}

func (b *RabbitBroker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.conn != nil && !b.conn.IsClosed()
}

func (b *RabbitBroker) Channel() (*amqp.Channel, error) {
	b.mu.RLock()
	conn := b.conn
	b.mu.RUnlock()

	if conn == nil || conn.IsClosed() {
		return nil, fmt.Errorf("rabbitmq: not connected")
	}
	return conn.Channel()
}

func (b *RabbitBroker) Close() error {
	b.mu.Lock()
	b.closed = true
	conn := b.conn
	b.conn = nil
	b.mu.Unlock()

	b.cancel()

	if conn != nil && !conn.IsClosed() {
		return conn.Close()
	}
	return nil
}

func (b *RabbitBroker) connectWithRetry(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("rabbitmq: broker is closed")
	}

	var lastErr error
	for i := 1; i <= b.cfg.MaxRetries; i++ {
		b.conn, lastErr = amqp.Dial(b.cfg.URL)
		if lastErr == nil {
			break
		}

		// Shared logger wrapper functions standard fields ke sath use ho rahe hain
		logging.Warn("rabbitmq connection attempt failed", map[string]any{
			"attempt": i,
			"max":     b.cfg.MaxRetries,
			"error":   lastErr.Error(),
		})

		if i < b.cfg.MaxRetries {
			select {
			case <-ctx.Done():
				return fmt.Errorf("rabbitmq: connect cancelled: %w", ctx.Err())
			case <-time.After(b.cfg.RetryDelay):
			}
		}
	}

	if lastErr != nil {
		return fmt.Errorf("rabbitmq: connection failed after %d attempts: %w", b.cfg.MaxRetries, lastErr)
	}

	logging.Info("rabbitmq connected", map[string]any{
		"host":  b.cfg.Host,
		"vhost": b.cfg.VHost,
	})

	return nil
}

func (b *RabbitBroker) watchConnection() {
	for {
		b.mu.RLock()
		conn := b.conn
		closed := b.closed
		b.mu.RUnlock()

		if closed || conn == nil {
			return
		}

		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))

		select {
		case <-b.ctx.Done():
			return
		case amqpErr, ok := <-closeCh:
			if !ok {
				return
			}
			if amqpErr != nil {
				logging.Error("rabbitmq connection dropped", amqpErr, nil)
			}
		}

		for {
			b.mu.RLock()
			isClosed := b.closed
			b.mu.RUnlock()

			if isClosed {
				return
			}

			if err := b.connectWithRetry(context.Background()); err == nil {
				logging.Info("rabbitmq reconnected", nil)
				break
			}

			logging.Warn("rabbitmq reconnect failed, retrying in 5s", nil)

			select {
			case <-time.After(5 * time.Second):
			case <-b.ctx.Done():
				return
			}
		}
	}
}