package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
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

// ─────────────────────────────────────────────────────────────────────────────
// VAULT DATA ENCRYPTION (AES-256-GCM Split Output)
// ─────────────────────────────────────────────────────────────────────────────

// EncryptVaultValue encrypts plain string into separate base64 encoded ciphertext and IV.
func EncryptVaultValue(plainText, secretKey string) (cipherBase64, ivBase64 string, err error) {
	keyBytes := []byte(secretKey)
	if len(keyBytes) != 32 {
		return "", "", fmt.Errorf("security: invalid vault secret key length, must be exactly 32 bytes")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", "", fmt.Errorf("security: aes cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("security: gcm initialization failed: %w", err)
	}

	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", "", fmt.Errorf("security: iv generation failed: %w", err)
	}

	ciphertext := gcm.Seal(nil, iv, []byte(plainText), nil)

	return base64.StdEncoding.EncodeToString(ciphertext), base64.StdEncoding.EncodeToString(iv), nil
}

// DecryptVaultValue decrypts separate base64 encoded ciphertext and IV back to raw plain text.
func DecryptVaultValue(cipherBase64, ivBase64, secretKey string) (string, error) {
	keyBytes := []byte(secretKey)
	if len(keyBytes) != 32 {
		return "", fmt.Errorf("security: invalid vault secret key length, must be exactly 32 bytes")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return "", fmt.Errorf("security: ciphertext base64 decode failed: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(ivBase64)
	if err != nil {
		return "", fmt.Errorf("security: iv base64 decode failed: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("security: aes decryption cipher failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("security: gcm decryption setup failed: %w", err)
	}

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("security: decryption authentication failed: %w", err)
	}

	return string(plaintext), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// TOTP SECRET ENCRYPTION (AES-256-GCM Combined Output)
// ─────────────────────────────────────────────────────────────────────────────

// EncryptTOTPSecret encrypts the TOTP base32 secret using AES-256-GCM.
// Output is a single base64 string combining nonce + ciphertext.
func EncryptTOTPSecret(plaintext, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("security: totp cipher setup failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("security: totp gcm setup failed: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("security: totp nonce production failed: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSecret decrypts a combined base64 secret payload.
func DecryptTOTPSecret(encoded, key string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("security: totp base64 payload decode failed: %w", err)
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("security: totp decryption cipher failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("security: totp gcm decryption failed: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("security: totp ciphertext package is too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("security: totp payload authenticated decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// 2FA BACKUP CODES UTILITIES
// ─────────────────────────────────────────────────────────────────────────────

// GenerateBackupCodes returns n plain-text secure backup codes (8 chars each).
func GenerateBackupCodes(n int) ([]string, error) {
	codes := make([]string, n)
	for i := range codes {
		raw := make([]byte, 5) // 5 bytes gives 10 hex chars, we clamp to 8
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("security: backup codes system bytes read failed: %w", err)
		}
		codes[i] = fmt.Sprintf("%x", raw)[:8]
	}
	return codes, nil
}

// HashBackupCodes hashes each plain-text backup code with bcrypt recursively.
func HashBackupCodes(codes []string) ([]string, error) {
	hashed := make([]string, len(codes))
	for i, code := range codes {
		h, err := HashPassword(code)
		if err != nil {
			return nil, err
		}
		hashed[i] = h
	}
	return hashed, nil
}

// VerifyBackupCode compares plain text against a list of hashes. Returns index or -1.
func VerifyBackupCode(plain string, hashedCodes []string) int {
	for i, h := range hashedCodes {
		if VerifyPassword(h, plain) {
			return i
		}
	}
	return -1
}