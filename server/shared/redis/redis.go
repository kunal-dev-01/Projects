package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"shared/logging" 

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr               string
	Host               string
	Port               int
	Password           string
	DB                 int
	MaxRetries         int
	PoolSize           int
	PoolTimeout        time.Duration
	MaxConnAge         time.Duration
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	RetryAttempts      int
	RetryDelay         time.Duration
}

type Redis struct {
	mu     sync.RWMutex
	client *redis.Client
	cfg    Config
	closed bool
}

func New(cfg Config) *Redis {
	return &Redis{cfg: cfg}
}

func (r *Redis) Connect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return fmt.Errorf("redis: instance is closed")
	}

	opts := &redis.Options{
		Addr:            r.cfg.Addr,
		Password:        r.cfg.Password,
		DB:              r.cfg.DB,
		PoolSize:        r.cfg.PoolSize,
		MinIdleConns:    max(r.cfg.PoolSize/4, 2),
		PoolTimeout:     r.cfg.PoolTimeout,
		ConnMaxLifetime: r.cfg.MaxConnAge,
		ConnMaxIdleTime: r.cfg.IdleTimeout,
		DialTimeout:     r.cfg.DialTimeout,
		ReadTimeout:     r.cfg.ReadTimeout,
		WriteTimeout:    r.cfg.WriteTimeout,
		MaxRetries:      r.cfg.MaxRetries,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		OnConnect: func(ctx context.Context, _ *redis.Conn) error {
			logging.Debug("redis new connection established", nil)
			return nil
		},
	}

	r.client = redis.NewClient(opts)

	var lastErr error
	for i := 1; i <= r.cfg.RetryAttempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		lastErr = r.client.Ping(pingCtx).Err()
		cancel()

		if lastErr == nil {
			logging.Info("redis connected", map[string]any{
				"host":      r.cfg.Host,
				"port":      r.cfg.Port,
				"db":        r.cfg.DB,
				"pool_size": r.cfg.PoolSize,
				"min_idle":  opts.MinIdleConns,
			})
			return nil
		}

		logging.Warn("redis ping failed", map[string]any{
			"attempt": i,
			"max":     r.cfg.RetryAttempts,
			"error":   lastErr.Error(),
		})

		if i < r.cfg.RetryAttempts {
			select {
			case <-ctx.Done():
				r.client = nil
				return fmt.Errorf("redis: connect cancelled: %w", ctx.Err())
			case <-time.After(r.cfg.RetryDelay):
			}
		}
	}

	r.client = nil
	return fmt.Errorf("redis: connection failed after %d attempts: %w", r.cfg.RetryAttempts, lastErr)
}

func (r *Redis) Close() error {
	r.mu.Lock()
	r.closed = true
	client := r.client
	r.client = nil
	r.mu.Unlock()

	if client == nil {
		return nil
	}

	if err := client.Close(); err != nil {
		logging.Error("redis close error", err, nil)
		return err
	}

	logging.Info("redis closed gracefully", nil)
	return nil
}

func (r *Redis) Client() (*redis.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return nil, fmt.Errorf("redis: instance is closed")
	}
	if r.client == nil {
		return nil, fmt.Errorf("redis: not connected")
	}
	return r.client, nil
}

func (r *Redis) IsConnected(ctx context.Context) bool {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	if client == nil {
		return false
	}
	return client.Ping(ctx).Err() == nil
}

func (r *Redis) Stats() *redis.PoolStats {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	if client == nil {
		return nil
	}
	return client.PoolStats()
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	client, err := r.Client()
	if err != nil {
		return "", err
	}

	val, err := client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

func (r *Redis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	client, err := r.Client()
	if err != nil {
		return err
	}
	return client.Set(ctx, key, value, ttl).Err()
}

func (r *Redis) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	client, err := r.Client()
	if err != nil {
		return false, err
	}
	return client.SetNX(ctx, key, value, ttl).Result()
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	client, err := r.Client()
	if err != nil {
		return err
	}
	return client.Del(ctx, keys...).Err()
}

func (r *Redis) Exists(ctx context.Context, keys ...string) (int64, error) {
	client, err := r.Client()
	if err != nil {
		return 0, err
	}
	return client.Exists(ctx, keys...).Result()
}

func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	client, err := r.Client()
	if err != nil {
		return 0, err
	}
	return client.Incr(ctx, key).Result()
}

func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	client, err := r.Client()
	if err != nil {
		return false, err
	}
	return client.Expire(ctx, key, ttl).Result()
}

func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	client, err := r.Client()
	if err != nil {
		return 0, err
	}
	return client.TTL(ctx, key).Result()
}