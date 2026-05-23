package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hookzen-server/internal/config" 

	sharedErrs "shared/errors"
	"shared/logging"
	sharedRedis "shared/redis"
)

const (
	prefixSession      = "auth:session:"
	prefixUserSessions = "auth:user_sessions:"
)

// SessionMetadata represents the unified production identity container.
type SessionMetadata struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	IP         string    `json:"ip"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	DeviceType string    `json:"device_type"`
	City       string    `json:"city"`
	Country    string    `json:"country"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type SessionRepository struct {
	r   *sharedRedis.Redis
	cfg *config.Config
}

func NewSessionRepository(r *sharedRedis.Redis, cfg *config.Config) *SessionRepository {
	return &SessionRepository{r: r, cfg: cfg}
}

// ─────────────────────────────────────────────────────────────────────────────
// CORE LIFE-CYCLE MUTATIONS
// ─────────────────────────────────────────────────────────────────────────────

// StoreSession checks allocation bounds from config and pipes session indexes atomically.
func (s *SessionRepository) StoreSession(ctx context.Context, meta *SessionMetadata, ttl time.Duration) error {
	if meta == nil {
		return sharedErrs.Internal(fmt.Errorf("session: metadata input cannot be nil"))
	}

	// Dynamic real-time count bounds verification
	count, err := s.CountActiveSessions(ctx, meta.UserID)
	if err != nil {
		return err
	}
	if count >= s.cfg.Security.MaxSessions {
		return sharedErrs.Forbidden("maximum active sessions reached — logout from another device first")
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return sharedErrs.Internal(err)
	}

	client, err := s.r.Client()
	if err != nil {
		return sharedErrs.Internal(err)
	}

	sessionKey := prefixSession + meta.ID
	userKey := s.buildUserKey(meta.UserID)
	userSetTTL := s.cfg.JWT.RefreshTokenDuration

	pipe := client.Pipeline()
	pipe.Set(ctx, sessionKey, data, ttl)
	pipe.SAdd(ctx, userKey, meta.ID)
	pipe.Expire(ctx, userKey, userSetTTL)

	if _, err = pipe.Exec(ctx); err != nil {
		logging.Error("session: atomicity layer execution pipeline crashed", err, map[string]any{"user_id": meta.UserID})
		return sharedErrs.Internal(err)
	}

	return nil
}

// GetSession extracts raw values and safely handles structure deserialization.
func (s *SessionRepository) GetSession(ctx context.Context, sessionID string) (*SessionMetadata, error) {
	raw, err := s.r.Get(ctx, prefixSession+sessionID)
	if err != nil {
		return nil, sharedErrs.Internal(err)
	}
	if raw == "" {
		return nil, sharedErrs.NotFound("session expired or invalidated")
	}

	var meta SessionMetadata
	if err = json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, sharedErrs.Internal(err)
	}

	return &meta, nil
}

// RevokeSession kicks out a targeted single execution link.
func (s *SessionRepository) RevokeSession(ctx context.Context, userID, sessionID string) error {
	client, err := s.r.Client()
	if err != nil {
		return sharedErrs.Internal(err)
	}

	pipe := client.Pipeline()
	pipe.Del(ctx, prefixSession+sessionID)
	pipe.SRem(ctx, s.buildUserKey(userID), sessionID)

	if _, err = pipe.Exec(ctx); err != nil {
		logging.Error("session: targeted token eviction failed", err, map[string]any{"session_id": sessionID})
		return sharedErrs.Internal(err)
	}

	return nil
}

// RevokeAllSessions clears out everything linked to this specific entity.
func (s *SessionRepository) RevokeAllSessions(ctx context.Context, userID string) error {
	client, err := s.r.Client()
	if err != nil {
		return sharedErrs.Internal(err)
	}

	userKey := s.buildUserKey(userID)
	sessionIDs, err := client.SMembers(ctx, userKey).Result()
	if err != nil {
		return sharedErrs.Internal(err)
	}
	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := client.Pipeline()
	for _, id := range sessionIDs {
		pipe.Del(ctx, prefixSession+id)
	}
	pipe.Del(ctx, userKey)

	if _, err = pipe.Exec(ctx); err != nil {
		logging.Error("session: master execution stack cleanup dropped", err, map[string]any{"user_id": userID})
		return sharedErrs.Internal(err)
	}

	return nil
}

// DeleteSession handles an absolute identity lookup cleanup sequentially.
func (s *SessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	meta, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.RevokeSession(ctx, meta.UserID, sessionID)
}

// ─────────────────────────────────────────────────────────────────────────────
// PARALLEL EVALUATION ENGINE & LAZY PRUNER
// ─────────────────────────────────────────────────────────────────────────────

// GetActiveSessions fetches profiles via MGet parallel sweeps and clears dirty rows asynchronously.
func (s *SessionRepository) GetActiveSessions(ctx context.Context, userID string) ([]*SessionMetadata, error) {
	client, err := s.r.Client()
	if err != nil {
		return nil, sharedErrs.Internal(err)
	}

	userKey := s.buildUserKey(userID)
	sessionIDs, err := client.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, sharedErrs.Internal(err)
	}
	if len(sessionIDs) == 0 {
		return []*SessionMetadata{}, nil
	}

	keys := make([]string, len(sessionIDs))
	for i, id := range sessionIDs {
		keys[i] = prefixSession + id
	}

	results, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, sharedErrs.Internal(err)
	}

	sessions := make([]*SessionMetadata, 0, len(sessionIDs))
	var staleIDs []any

	for i, res := range results {
		if res == nil {
			staleIDs = append(staleIDs, sessionIDs[i])
			continue
		}

		raw, ok := res.(string)
		if !ok {
			staleIDs = append(staleIDs, sessionIDs[i])
			continue
		}

		var meta SessionMetadata
		if err = json.Unmarshal([]byte(raw), &meta); err != nil {
			logging.Error("session: unmarshalling structural corruption trace — pruning entry", err, map[string]any{"session_id": sessionIDs[i]})
			staleIDs = append(staleIDs, sessionIDs[i])
			continue
		}

		sessions = append(sessions, &meta)
	}

	// Asynchronous decoupling for lazy garbage indexes removal (Zero latency on hot paths)
	if len(staleIDs) > 0 {
		go func(items []any) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = client.SRem(bgCtx, userKey, items...).Err()
		}(staleIDs)
	}

	return sessions, nil
}

func (s *SessionRepository) CountActiveSessions(ctx context.Context, userID string) (int, error) {
	sessions, err := s.GetActiveSessions(ctx, userID)
	if err != nil {
		return 0, err
	}
	return len(sessions), nil
}

func (s *SessionRepository) buildUserKey(userID string) string {
	return prefixUserSessions + userID
}