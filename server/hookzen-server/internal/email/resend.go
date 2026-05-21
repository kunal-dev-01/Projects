package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hookzen-server/internal/config"
	"shared/logging"
)

type EmailType string

const (
	EmailVerify EmailType = "verify_email"
	EmailReset  EmailType = "password_reset"
)

type EmailPayload struct {
	To      string         `json:"to"`
	Subject string         `json:"subject"`
	Type    EmailType      `json:"type"`
	Data    map[string]any `json:"data"`
}

type ResendClient struct {
	apiKey    string
	fromEmail string
	client    *http.Client
}

func NewResendClient(cfg config.ResendConfig) *ResendClient {
	return &ResendClient{
		apiKey:    cfg.APIKey,
		fromEmail: cfg.FromEmail,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (c *ResendClient) SendEmail(to, subject, html string) error {
	url := "https://api.resend.com/emails"

	reqBody := resendRequest{
		From:    c.fromEmail,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("email: failed to marshal resend payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("email: failed to build http request descriptor: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("email: network transportation to resend api failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var resErr any
		_ = json.NewDecoder(resp.Body).Decode(&resErr)
		
		logging.Error("email: resend production outbound dispatch error received", fmt.Errorf("status: %d", resp.StatusCode), map[string]any{
			"response_payload": resErr,
			"target_recipient": to,
		})
		return fmt.Errorf("email: resend provider rejected execution with status %d", resp.StatusCode)
	}

	return nil
}