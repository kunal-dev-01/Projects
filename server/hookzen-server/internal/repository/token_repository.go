package repository

import (
	"context"
	"fmt"
	"time"

	sharedErrs "shared/errors"
	"shared/logging"
	sharedRedis "shared/redis"
)

// ─────────────────────────────────────────────────────────────────────────────
// HOOKZEN SYSTEM TOKEN PREFIXES
// ─────────────────────────────────────────────────────────────────────────────

const (
	prefixEmailVerify   = "auth:token:email_verify:"   // auth:token:email_verify:<token> -> user_id
	prefixPasswordReset = "auth:token:pwd_reset:"      // auth:token:pwd_reset:<token>    -> user_id
	prefixTwoFASetup    = "auth:token:2fa_setup:"      // auth:token:2fa_setup:<user_id>  -> temp_secret_bytes (Settings me)
	prefixTwoFAPending  = "auth:token:2fa_pending:"    // auth:token:2fa_pending:<token>  -> user_id (Login ke waqt)
	prefixLoginAttempt  = "auth:security:attempt:"     // auth:security:attempt:<email>   -> count
	prefixAccountLock   = "auth:security:lock:"        // auth:security:lock:<email>      -> "1"
)

type TokenRepository struct {
	r *sharedRedis.Redis
}

func NewTokenRepository(r *sharedRedis.Redis) *TokenRepository {
	return &TokenRepository{r: r}
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. EMAIL VERIFICATION LIFECYCLE
// ─────────────────────────────────────────────────────────────────────────────

func (t *TokenRepository) StoreEmailVerifyToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	ok, err := t.r.SetNX(ctx, prefixEmailVerify+token, userID, ttl)
	if err != nil {
		logging.Error("token: failed to write email verification mapping", err, map[string]any{"user_id": userID})
		return sharedErrs.Internal(err)
	}
	if !ok {
		return sharedErrs.Conflict("verification token already active, please request a fresh link")
	}
	return nil
}

func (t *TokenRepository) ConsumeEmailVerifyToken(ctx context.Context, token string) (string, error) {
	return t.consumeToken(ctx, prefixEmailVerify+token, "email verification")
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. PASSWORD RESET LIFECYCLE
// ─────────────────────────────────────────────────────────────────────────────

func (t *TokenRepository) StorePasswordResetToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	ok, err := t.r.SetNX(ctx, prefixPasswordReset+token, userID, ttl)
	if err != nil {
		logging.Error("token: failed to write password reset mapping", err, map[string]any{"user_id": userID})
		return sharedErrs.Internal(err)
	}
	if !ok {
		return sharedErrs.Conflict("password reset token already active, please request a fresh link")
	}
	return nil
}

func (t *TokenRepository) ConsumePasswordResetToken(ctx context.Context, token string) (string, error) {
	return t.consumeToken(ctx, prefixPasswordReset+token, "password reset")
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. 2FA SETUP PENDING (Settings Layer — User Is Already Logged In)
// ─────────────────────────────────────────────────────────────────────────────

// StoreTwoFASetupSecret stores encrypted TOTP secret keyed by userID until verified. TTL: 10-15m
func (t *TokenRepository) StoreTwoFASetupSecret(ctx context.Context, userID, encryptedSecret string, ttl time.Duration) error {
	key := prefixTwoFASetup + userID
	err := t.r.Set(ctx, key, encryptedSecret, ttl)
	if err != nil {
		logging.Error("token: failed to cache temporary 2fa setup secret", err, map[string]any{"user_id": userID})
		return sharedErrs.Internal(err)
	}
	return nil
}

// GetTwoFASetupSecret retrieves the temporary secret for initial code validation check.
func (t *TokenRepository) GetTwoFASetupSecret(ctx context.Context, userID string) (string, error) {
	key := prefixTwoFASetup + userID
	secret, err := t.r.Get(ctx, key)
	if err != nil {
		return "", sharedErrs.Internal(err)
	}
	if secret == "" {
		return "", sharedErrs.NotFound("2fa activation window expired, please start setup again")
	}
	return secret, nil
}

// ClearTwoFASetupSecret deletes the staging setup key instantly after activation or abort.
func (t *TokenRepository) ClearTwoFASetupSecret(ctx context.Context, userID string) error {
	return t.r.Del(ctx, prefixTwoFASetup+userID)
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. 2FA LOGIN PENDING (Auth Flow Layer — User Logging In)
// ─────────────────────────────────────────────────────────────────────────────

func (t *TokenRepository) StoreTwoFAPendingToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	ok, err := t.r.SetNX(ctx, prefixTwoFAPending+token, userID, ttl)
	if err != nil {
		logging.Error("token: failed to register 2fa staging token", err, map[string]any{"user_id": userID})
		return sharedErrs.Internal(err)
	}
	if !ok {
		return sharedErrs.Conflict("an active authentication challenge is already in progress")
	}
	return nil
}

func (t *TokenRepository) ConsumeTwoFAPendingToken(ctx context.Context, token string) (string, error) {
	return t.consumeToken(ctx, prefixTwoFAPending+token, "2fa staging")
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. BRUTE-FORCE SECURITY DEFENCE SYSTEMS
// ─────────────────────────────────────────────────────────────────────────────

func (t *TokenRepository) IncrLoginAttempt(ctx context.Context, email string, windowTTL time.Duration) (int64, error) {
	key := prefixLoginAttempt + email
	count, err := t.r.Incr(ctx, key)
	if err != nil {
		return 0, sharedErrs.Internal(err)
	}

	if count == 1 {
		if _, err = t.r.Expire(ctx, key, windowTTL); err != nil {
			logging.Error("token: failed to set login attempt tracking window expiration", err, map[string]any{"email": email})
		}
	}
	return count, nil
}

func (t *TokenRepository) LockAccount(ctx context.Context, email string, lockDuration time.Duration) error {
	err := t.r.Set(ctx, prefixAccountLock+email, "1", lockDuration)
	if err != nil {
		logging.Error("token: account lockout commit failed", err, map[string]any{"email": email})
		return sharedErrs.Internal(err)
	}
	return nil
}

func (t *TokenRepository) IsAccountLocked(ctx context.Context, email string) (bool, error) {
	val, err := t.r.Get(ctx, prefixAccountLock+email)
	if err != nil {
		return false, sharedErrs.Internal(err)
	}
	return val == "1", nil
}

func (t *TokenRepository) ClearLoginAttempts(ctx context.Context, email string) error {
	return t.r.Del(ctx, prefixLoginAttempt+email, prefixAccountLock+email)
}

// ─────────────────────────────────────────────────────────────────────────────
// INTERNAL UTILS
// ─────────────────────────────────────────────────────────────────────────────

func (t *TokenRepository) consumeToken(ctx context.Context, key, label string) (string, error) {
	userID, err := t.r.Get(ctx, key)
	if err != nil {
		return "", sharedErrs.Internal(err)
	}
	if userID == "" {
		return "", sharedErrs.Unauthorized(fmt.Sprintf("the requested %s token is invalid or has expired", label))
	}

	if err = t.r.Del(ctx, key); err != nil {
		logging.Error("token: background destruction tracking leak detected", err, map[string]any{"key": key})
	}

	return userID, nil
}