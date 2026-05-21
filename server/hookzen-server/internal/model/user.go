package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// CUSTOM TYPES & ENUMS
// ─────────────────────────────────────────────────────────────────────────────

type TwoFactorMethod string

const (
	TwoFactorMethodTOTP       TwoFactorMethod = "totp"
	TwoFactorMethodBackupCode TwoFactorMethod = "backup_code"
)

// WebhookNotifications handles the flat JSONB structured configurations natively.
type WebhookNotifications struct {
	DeliveryFailures  bool `json:"delivery_failures"`
	EndpointDisabled  bool `json:"endpoint_disabled"`
	RateLimitWarnings bool `json:"rate_limit_warnings"`
	SecurityAlerts    bool `json:"security_alerts"`
}

// Scan implements the sql.Scanner interface for pgx JSONB automated decoding.
func (wn *WebhookNotifications) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("models: type assertion to []byte failed for WebhookNotifications")
	}
	return json.Unmarshal(b, wn)
}

// Value implements the driver.Valuer interface for pgx JSONB automated encoding.
func (wn WebhookNotifications) Value() (driver.Value, error) {
	return json.Marshal(wn)
}

// ─────────────────────────────────────────────────────────────────────────────
// CORE DATABASE MODELS
// ─────────────────────────────────────────────────────────────────────────────

// HookzenUser represents the 'hookzen_users' entity table.
type HookzenUser struct {
	ID                       uuid.UUID  `json:"id" db:"id"`
	Email                    string     `json:"email" db:"email"`
	EmailVerified            bool       `json:"email_verified" db:"email_verified"`
	EmailVerifiedAt          *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`
	PasswordHash             string     `json:"-" db:"password_hash"`
	PasswordChangedAt        *time.Time `json:"password_changed_at,omitempty" db:"password_changed_at"`
	PasswordResetRequestedAt *time.Time `json:"password_reset_requested_at,omitempty" db:"password_reset_requested_at"`
	FullName                 string     `json:"full_name" db:"full_name"`
	Timezone                 string     `json:"timezone" db:"timezone"`
	Locale                   string     `json:"locale" db:"locale"`
	IsActive                 bool       `json:"is_active" db:"is_active"`
	LastLoginAt              *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	LastLoginIP              *string    `json:"last_login_ip,omitempty" db:"last_login_ip"`
	TwoFactorEnabled         bool       `json:"two_factor_enabled" db:"two_factor_enabled"`
	TwoFactorEnabledAt       *time.Time `json:"two_factor_enabled_at,omitempty" db:"two_factor_enabled_at"`
	CreatedAt                time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt                *time.Time `json:"-" db:"deleted_at"`
}

// TwoFactorAuth represents the 'two_factor_auth' entity table.
type TwoFactorAuth struct {
	ID                      uuid.UUID        `json:"id" db:"id"`
	UserID                  uuid.UUID        `json:"user_id" db:"user_id"`
	TOTPSecret              string           `json:"-" db:"totp_secret"`
	TOTPAlgorithm           string           `json:"totp_algorithm" db:"totp_algorithm"`
	TOTPDigits              int16            `json:"totp_digits" db:"totp_digits"`
	TOTPPeriod              int16            `json:"totp_period" db:"totp_period"`
	BackupCodes             []string         `json:"-" db:"backup_codes"`
	BackupCodesGeneratedAt *time.Time       `json:"backup_codes_generated_at,omitempty" db:"backup_codes_generated_at"`
	IsEnabled               bool             `json:"is_enabled" db:"is_enabled"`
	EnabledAt               *time.Time       `json:"enabled_at,omitempty" db:"enabled_at"`
	LastVerifiedAt          *time.Time       `json:"last_verified_at,omitempty" db:"last_verified_at"`
	LastMethodUsed          *TwoFactorMethod `json:"last_method_used,omitempty" db:"last_method_used"`
	CreatedAt               time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time        `json:"updated_at" db:"updated_at"`
}

// UserPreferences represents the 'user_preferences' entity table.
type UserPreferences struct {
	UserID        uuid.UUID            `json:"user_id" db:"user_id"`
	Notifications WebhookNotifications `json:"notifications" db:"notifications"`
	Language      string               `json:"language" db:"language"`
	UpdatedAt     time.Time            `json:"updated_at" db:"updated_at"`
}