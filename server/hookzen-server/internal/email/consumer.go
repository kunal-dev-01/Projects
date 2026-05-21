package email

import (
	"context"
	"encoding/json"
	"time"

	"hookzen-server/internal/config"
	"shared/broker"
	"shared/logging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EmailConsumer struct {
	broker  *broker.RabbitBroker
	service *EmailService
	cfg     *config.Config
}

func NewEmailConsumer(b *broker.RabbitBroker, s *EmailService, cfg *config.Config) *EmailConsumer {
	return &EmailConsumer{
		broker:  b,
		service: s,
		cfg:     cfg,
	}
}

func (c *EmailConsumer) Start(ctx context.Context) error {
	ch, err := c.broker.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		c.cfg.RabbitMQ.EmailQueue, // dynamic queue target representation
		true,                      // durable memory persistence states bound
		false,                     // clean mapping configuration parameters
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		c.cfg.RabbitMQ.EmailQueue,
		"email-service-worker",    // discrete explicit subscriber identifier
		false,                     // manual ack handling enforcement for transactional guarantees
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	logging.Info("email task async consumer initialized on target broker", map[string]any{"queue": c.cfg.RabbitMQ.EmailQueue})

	for {
		select {
		case <-ctx.Done():
			logging.Info("email task consumer shutdown command received, releasing attachments", nil)
			return nil
		case d, ok := <-msgs:
			if !ok {
				logging.Warn("email execution pathway terminated unexpectedly, target amqp channel dropped", nil)
				return nil
			}
			// Safe concurrent multi-threaded task spawning boundaries
			go c.handleDelivery(d)
		}
	}
}

func (c *EmailConsumer) handleDelivery(d amqp.Delivery) {
	var payload EmailPayload
	if err := json.Unmarshal(d.Body, &payload); err != nil {
		logging.Error("email: corrupted payload mapping drop tracking block, unmarshal failed", err, nil)
		_ = d.Ack(false) // Dead poison message cleanly evacuated from pipe
		return
	}

	var lastErr error
	maxRetries := c.cfg.RabbitMQ.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for i := 0; i < maxRetries; i++ {
		if err := c.service.ProcessEmail(payload); err == nil {
			logging.Debug("email worker: outbound task dispatched completely", map[string]any{
				"to":   payload.To,
				"type": payload.Type,
			})
			_ = d.Ack(false) // Safe clean completion handshake
			return
		} else {
			lastErr = err
			logging.Warn("email worker: dispatch attempt failed, initializing retry procedures", map[string]any{
				"to":      payload.To,
				"attempt": i + 1,
				"max":     maxRetries,
				"error":   err.Error(),
			})
			
			// Dynamic configurable linear/multiplier retry backoff interval handler
			backoffInterval := time.Duration(i+1) * c.cfg.RabbitMQ.RetryDelay
			if backoffInterval <= 0 {
				backoffInterval = 2 * time.Second // Fail-safe default
			}
			time.Sleep(backoffInterval)
		}
	}

	logging.Error("email worker: transactional message drop, threshold limit bounds passed", lastErr, map[string]any{
		"to":   payload.To,
		"type": payload.Type,
	})
	
	// Final execution escape acknowledgment to prevent infinite circular loops inside single broker queues.
	// Production best-practice standard: Map dead letter exchange hooks inside exchange topology configurations.
	_ = d.Ack(false)
}