package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	sharedErrs "shared/errors"
	sharedSecurity "shared/security"
)

// ─────────────────────────────────────────────────────────────────────────────
// AES-256-GCM TARGET SECRETS HANDLING
// ─────────────────────────────────────────────────────────────────────────────

// EncryptTOTPSecret encodes raw base32 metadata tokens natively using an atomic cipher system.
func EncryptTOTPSecret(plaintext, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", sharedErrs.Internal(fmt.Errorf("security: multi-factor aes block construction error: %w", err))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", sharedErrs.Internal(fmt.Errorf("security: multi-factor context gcm allocation failed: %w", err))
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", sharedErrs.Internal(fmt.Errorf("security: multi-factor secure randomness system stream corrupted: %w", err))
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSecret evaluates verification blocks to safely extract backing seed structures.
func DecryptTOTPSecret(encoded, key string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", sharedErrs.Internal(fmt.Errorf("security: base64 payload decoding fault: %w", err))
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", sharedErrs.Internal(fmt.Errorf("security: verification cipher validation failure: %w", err))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", sharedErrs.Internal(fmt.Errorf("security: decryption block setup failure: %w", err))
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", sharedErrs.Internal(fmt.Errorf("security: decryption processing constraint truncated"))
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", sharedErrs.Internal(fmt.Errorf("security: multi-factor authentication payload verification failed: %w", err))
	}

	return string(plaintext), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RECOVERY BACKUP MANAGEMENT
// ─────────────────────────────────────────────────────────────────────────────

// GenerateBackupCodes returns a set of unique cryptographically random alphanumeric backup trackers.
func GenerateBackupCodes(n int) ([]string, error) {
	codes := make([]string, n)
	for i := range codes {
		raw := make([]byte, 5) // 5 secure random bytes mapping into a clean length clamp
		if _, err := rand.Read(raw); err != nil {
			return nil, sharedErrs.Internal(fmt.Errorf("security: recovery subsystem randomness engine failed: %w", err))
		}
		codes[i] = fmt.Sprintf("%x", raw)[:8]
	}
	return codes, nil
}

// HashBackupCodes processes each plain code string using native Bcrypt algorithms securely.
func HashBackupCodes(codes []string) ([]string, error) {
	hashed := make([]string, len(codes))
	for i, code := range codes {
		h, err := sharedSecurity.HashPassword(code) // Calls our clean HashPassword logic
		if err != nil {
			return nil, err
		}
		hashed[i] = h
	}
	return hashed, nil
}

// VerifyBackupCode loops through existing slices to validate entry matching points. Returns trace index or -1.
func VerifyBackupCode(plain string, hashedCodes []string) int {
	for i, h := range hashedCodes {
		if sharedSecurity.VerifyPassword(h, plain) {
			return i
		}
	}
	return -1
}