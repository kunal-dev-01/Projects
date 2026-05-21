package dto

import (
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// REQUEST PAYLOADS (With Custom Strict Validation Tags)
// ─────────────────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,secure_password"`
	FullName string `json:"full_name" validate:"required,valid_name"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required,len=64"` // Enforces secure 64-char hex tokens
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,secure_password"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// ─────────────────────────────────────────────────────────────────────────────
// NESTED PREFERENCES RESPONSES (Webhook Gateway Metrics)
// ─────────────────────────────────────────────────────────────────────────────

type WebhookNotificationsResponse struct {
	DeliveryFailures  bool `json:"delivery_failures"`
	EndpointDisabled  bool `json:"endpoint_disabled"`
	RateLimitWarnings bool `json:"rate_limit_warnings"`
	SecurityAlerts    bool `json:"security_alerts"`
}

type UserPreferencesResponse struct {
	Notifications WebhookNotificationsResponse `json:"notifications"`
	Language      string                       `json:"language"`
}

// ─────────────────────────────────────────────────────────────────────────────
// RESPONSE CONTRACTS
// ─────────────────────────────────────────────────────────────────────────────

// UserProfileResponse includes embedded preferences for direct frontend state hydration.
type UserProfileResponse struct {
	ID               uuid.UUID               `json:"id"`
	Email            string                  `json:"email"`
	EmailVerified    bool                    `json:"email_verified"`
	FullName         string                  `json:"full_name"`
	Timezone         string                  `json:"timezone"`
	Locale           string                  `json:"locale"`
	TwoFactorEnabled bool                    `json:"two_factor_enabled"`
	Preferences      UserPreferencesResponse `json:"preferences"` // Integrated preferences schema
	CreatedAt        time.Time               `json:"created_at"`
}

// AuthResponse is returned upon successful authentication (Refresh token goes to HttpOnly cookie).
type AuthResponse struct {
	AccessToken string              `json:"access_token"`
	User        UserProfileResponse `json:"user"`
}

// RegisterResponse represents the immediate response contract after registration.
type RegisterResponse struct {
	Message string `json:"message"`
}

// ─────────────────────────────────────────────────────────────────────────────
// 2FA REQUEST/RESPONSE PAYLOADS
// ─────────────────────────────────────────────────────────────────────────────

// Setup2FAResponse returns details needed to connect an Authenticator App.
type Setup2FAResponse struct {
	Secret      string   `json:"secret"`       // Plain Base32 secret shown to user if QR fails
	QRCodeURI   string   `json:"qr_code_uri"`  // otpauth:// top-level registration URI
	BackupCodes []string `json:"backup_codes"` // Plain text backup codes shown ONLY ONCE during setup
}

// Enable2FARequest is used to finalize and activate 2FA after checking the first code.
type Enable2FARequest struct {
	Code string `json:"code" validate:"required,numeric,len=6"` // Validates user scanned it right
}

// Verify2FALoginRequest handles both TOTP or Backup Code during multi-factor login challenges.
type Verify2FALoginRequest struct {
	Code   string `json:"code" validate:"required"`                             // Can be 6-digit TOTP or 8-char backup code
	Method string `json:"method" validate:"required,oneof=totp backup_code"` // Strict enum boundary validation
}

// TwoFactorStateResponse reports current MFA statuses on user profiles.
type TwoFactorStateResponse struct {
	IsEnabled       bool   `json:"is_enabled"`
	Message         string `json:"message"`
	BackupRemaining int    `json:"backup_remaining,omitempty"`
}