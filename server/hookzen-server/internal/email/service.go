package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"path/filepath"

	"hookzen-server/internal/config"
	"shared/broker"
	sharedErrs "shared/errors"
	"shared/logging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EmailService struct {
	cfg    *config.Config
	broker *broker.RabbitBroker
	resend *ResendClient
}

func NewEmailService(cfg *config.Config, b *broker.RabbitBroker) *EmailService {
	return &EmailService{
		cfg:    cfg,
		broker: b,
		resend: NewResendClient(cfg.Resend),
	}
}

func (s *EmailService) SendVerificationEmail(ctx context.Context, to, token string) error {
	verifyLink := fmt.Sprintf("%s?token=%s", s.cfg.Frontend.VerificationUrl, token)

	payload := EmailPayload{
		To:      to,
		Subject: "Verify your Hookzen account",
		Type:    EmailVerify,
		Data: map[string]any{
			"verify_link": verifyLink,
			"id":          token[:8],
		},
	}

	return s.publish(ctx, payload)
}

func (s *EmailService) SendPasswordResetEmail(ctx context.Context, to, token string) error {
	resetLink := fmt.Sprintf("%s?token=%s", s.cfg.Frontend.ResetUrl, token)

	payload := EmailPayload{
		To:      to,
		Subject: "Reset your Hookzen password",
		Type:    EmailReset,
		Data: map[string]any{
			"reset_link": resetLink,
			"request_id": token[:8],
		},
	}

	return s.publish(ctx, payload)
}

func (s *EmailService) publish(ctx context.Context, payload EmailPayload) error {
	ch, err := s.broker.Channel()
	if err != nil {
		return sharedErrs.Internal(err)
	}
	defer ch.Close()

	body, err := json.Marshal(payload)
	if err != nil {
		return sharedErrs.Internal(err)
	}

	err = ch.PublishWithContext(ctx,
		"",                        // default exchange mapping
		s.cfg.RabbitMQ.EmailQueue, // direct routing key target
		false,                     // mandatory configuration check
		false,                     // immediate transmission rules
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		logging.Error("email: exchange task production write rejected by rabbitmq", err, map[string]any{"to": payload.To})
		return sharedErrs.Internal(err)
	}

	return nil
}

func (s *EmailService) ProcessEmail(payload EmailPayload) error {
	templateName := string(payload.Type) + ".html"
	tmplPath := filepath.Join("internal", "templates", templateName)

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("email: structural formatting of file template %s failed: %w", templateName, err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, payload.Data); err != nil {
		return fmt.Errorf("email: internal runtime context buffer mapping execution failed for %s: %w", templateName, err)
	}

	return s.resend.SendEmail(payload.To, payload.Subject, body.String())
}