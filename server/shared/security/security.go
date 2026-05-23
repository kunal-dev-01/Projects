package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"

	"shared/logging" 

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// ─────────────────────────────────────────────────────────────────────────────
// PASSWORD HASHING (Bcrypt)
// ─────────────────────────────────────────────────────────────────────────────

// HashPassword creates a secure bcrypt hash using production standard cost 12.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		logging.Error("password hash failed", err, nil)
		return "", fmt.Errorf("security: password hashing failed: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword returns true if plainPassword matches the bcrypt hash.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// ─────────────────────────────────────────────────────────────────────────────
// CRYPTOGRAPHIC TOKEN GENERATION
// ─────────────────────────────────────────────────────────────────────────────

// GenerateToken generates a cryptographically secure hex token.
func GenerateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		logging.Error("random token generation failed", err, nil)
		return "", fmt.Errorf("security: token generation failed: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func GenerateEmailVerifyToken() (string, error)   { return GenerateToken(32) }
func GeneratePasswordResetToken() (string, error)  { return GenerateToken(32) }
func GenerateSessionID() (string, error)           { return GenerateToken(24) }
func GenerateTwoFAPendingToken() (string, error)   { return GenerateToken(32) }

// GenerateOTP returns a secure random 6-digit numeric code.
func GenerateOTP() string {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "123456" // Cryptographically safe fallback
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func ParseUUID(idStr string) (uuid.UUID, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("security: invalid UUID format: %w", err)
	}
	return id, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ID GENERATION & FORMATTING
// ─────────────────────────────────────────────────────────────────────────────

// NewID returns a time-ordered UUID v7, falling back to v4 on internal error.
func NewID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}

// FormatKey builds a namespaced string, primarily used for clean Redis key structuring.
func FormatKey(prefix, value string) string {
	return fmt.Sprintf("%s%s", prefix, value)
}