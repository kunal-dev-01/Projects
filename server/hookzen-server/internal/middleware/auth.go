package middleware

import (
	"hookzen-server/internal/repository"
	"hookzen-server/internal/jwt"

	sharedErrs "shared/errors"
	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware struct {
	jwt *jwt.JWTManager
	sess *repository.SessionRepository
}

func NewAuthMiddleware(jwt *jwt.JWTManager, sess *repository.SessionRepository) *AuthMiddleware {
	return &AuthMiddleware{
		jwt:  jwt,
		sess: sess,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. STANDARD AUTHENTICATED CORE INTERCEPTOR
// ─────────────────────────────────────────────────────────────────────────────

// RequireAuth protects core endpoints, ensuring full access token and active session validity.
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Extract Bearer token string from headers
		authHeader := c.Get("Authorization")
		tokenStr, err := jwt.ExtractToken(authHeader)
		if err != nil {
			return err // sharedErrs.Unauthorized standard propagation
		}

		// 2. Cryptographically validate access token signatures
		claims, err := m.jwt.ValidateToken(tokenStr, jwt.TokenAccess)
		if err != nil {
			return err
		}

		// 3. Stateful checkpoint: Verify session state bounds inside Redis cache
		ctx := c.Context()
		sessionID := claims.RegisteredClaims.ID
		
		_, err = m.sess.GetSession(ctx, sessionID)
		if err != nil {
			// If session was revoked or flushed from Redis, throw 401 instantly
			return sharedErrs.Unauthorized("session has been invalidated or forced logged out")
		}

		// 4. Hydrate Fiber Context Locals for logs integration and subsequent downstream handlers
		c.Locals("user_id", claims.UserID.String())
		c.Locals("session_id", sessionID)

		return c.Next()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. 2FA PENDING TRANSIT INTERCEPTOR
// ─────────────────────────────────────────────────────────────────────────────

// Require2FAPending restricts routes (like /auth/verify-2fa) to intermediate login challenge states.
func (m *AuthMiddleware) Require2FAPending() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		tokenStr, err := jwt.ExtractToken(authHeader)
		if err != nil {
			return err
		}

		// Strictly demands transient TokenTwoFAPending state token scope
		claims, err := m.jwt.ValidateToken(tokenStr, jwt.TokenTwoFAPending)
		if err != nil {
			return err
		}

		// Validate that the staging session token hasn't dropped off
		ctx := c.Context()
		_, err = m.sess.GetSession(ctx, claims.RegisteredClaims.ID)
		if err != nil {
			return sharedErrs.Unauthorized("auth staging token session expired")
		}

		c.Locals("pending_user_id", claims.UserID.String())
		c.Locals("pending_session_id", claims.RegisteredClaims.ID)

		return c.Next()
	}
}