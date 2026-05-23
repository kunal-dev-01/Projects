package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"hookzen-server/internal/config"
	"hookzen-server/internal/model"

	sharedErrs "shared/errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	TokenAccess       TokenType = "access"
	TokenRefresh      TokenType = "refresh"
	TokenTwoFAPending TokenType = "2fa_pending"
)

// CustomClaims represents the streamlined compact Hookzen JWT structure.
type CustomClaims struct {
	UserID           uuid.UUID `json:"uid"`
	Email            string    `json:"eml"`
	TwoFactorEnabled bool      `json:"2fa"`
	Type             TokenType `json:"typ"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type JWTManager struct {
	cfg           *config.JWTConfig
	secretBytes   []byte
}

func NewJWTManager(cfg *config.JWTConfig) *JWTManager {
	return &JWTManager{
		cfg:           cfg,
		secretBytes:   []byte(cfg.Secret),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CRYPTOGRAPHIC TOKEN GENERATION
// ─────────────────────────────────────────────────────────────────────────────

func (m *JWTManager) GenerateTokenPair(user *model.HookzenUser, sessionID string) (*TokenPair, error) {
	accessToken, err := m.generate(user.ID, user.Email, user.TwoFactorEnabled, TokenAccess, m.cfg.AccessTokenDuration, sessionID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.generate(user.ID, user.Email, user.TwoFactorEnabled, TokenRefresh, m.cfg.RefreshTokenDuration, sessionID)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(m.cfg.AccessTokenDuration).Unix(),
	}, nil
}

func (m *JWTManager) GenerateTwoFAPendingToken(user *model.HookzenUser, sessionID string) (string, error) {
	return m.generate(user.ID, user.Email, user.TwoFactorEnabled, TokenTwoFAPending, 5*time.Minute, sessionID)
}

func (m *JWTManager) generate(userID uuid.UUID, email string, twoFactor bool, tokenType TokenType, duration time.Duration, sessionID string) (string, error) {
	now := time.Now()

	claims := CustomClaims{
		UserID:           userID,
		Email:            email,
		TwoFactorEnabled: twoFactor,
		Type:             tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.cfg.Issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        sessionID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secretBytes)
	if err != nil {
		return "", sharedErrs.Internal(err)
	}

	return signed, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// PARSING & VERIFICATION INTERCEPTORS
// ─────────────────────────────────────────────────────────────────────────────

func (m *JWTManager) ValidateToken(tokenStr string, expectedType TokenType) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing mechanism allocation: %v", t.Header["alg"])
		}
		return m.secretBytes, nil
	})

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, sharedErrs.Unauthorized("token verification barrier: signature expired")
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, sharedErrs.Unauthorized("token verification barrier: invalid signature trace")
		default:
			return nil, sharedErrs.Unauthorized("token verification barrier: malformed payload packet")
		}
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, sharedErrs.Unauthorized("token verification barrier: claims context invalidated")
	}

	if claims.Type != expectedType {
		return nil, sharedErrs.Unauthorized(fmt.Sprintf("token verification barrier: expected %s scope mismatch", expectedType))
	}

	if claims.Issuer != m.cfg.Issuer {
		return nil, sharedErrs.Unauthorized("token verification barrier: untrusted structural authority issuer")
	}

	return claims, nil
}

func ExtractToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", sharedErrs.Unauthorized("missing authorization tracking token context header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return "", sharedErrs.Unauthorized("invalid authorization payload alignment format")
	}

	return parts[1], nil
}
