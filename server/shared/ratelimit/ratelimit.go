package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sharedErrs "shared/errors"
	"shared/logging"
	sharedRedis "shared/redis"

	"github.com/gofiber/fiber/v2"
)

type LimitConfig struct {
	Max      int
	Window   time.Duration
	KeyFunc  func(*fiber.Ctx) string
	Message  string
	SkipFunc func(*fiber.Ctx) bool
}

const luaIncrExpire = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`

// RateLimiter evaluates inbound request thresholds using atomic Redis Lua executions.
func RateLimiter(r *sharedRedis.Redis, cfg LimitConfig) fiber.Handler {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = ipKey
	}
	if cfg.Message == "" {
		cfg.Message = "too many requests, please slow down"
	}

	windowSecs := int64(cfg.Window.Seconds())

	client, err := r.Client()
	if err != nil {
		logging.Error("rate limiter: redis client unavailable at initialization", err, nil)
	}

	var scriptSHA string
	if client != nil {
		sha, err := client.ScriptLoad(context.Background(), luaIncrExpire).Result()
		if err != nil {
			logging.Error("rate limiter: lua script hash storage optimization failed", err, nil)
		} else {
			scriptSHA = sha
		}
	}

	return func(c *fiber.Ctx) error {
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			return c.Next()
		}

		cl, err := r.Client()
		if err != nil {
			logging.Error("rate limiter: client execution dropped (redis offline)", err, nil)
			return c.Next() // Fail open gracefully
		}

		slot := time.Now().Unix() / windowSecs
		key := fmt.Sprintf("rl:%s:%d", cfg.KeyFunc(c), slot)
		ctx := c.Context()

		res, err := cl.EvalSha(ctx, scriptSHA, []string{key}, strconv.FormatInt(windowSecs, 10)).Int64()
		if err != nil {
			logging.Error("rate limiter: atomic lua evaluation aborted", err, map[string]any{"key": key})
			return c.Next() // Fail open gracefully
		}

		count := int(res)
		remaining := max(cfg.Max-count, 0)

		ttl, _ := cl.TTL(ctx, key).Result()
		resetAt := time.Now().Add(ttl).Unix()

		c.Set("X-RateLimit-Limit", strconv.Itoa(cfg.Max))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if count > cfg.Max {
			c.Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))
			return sharedErrs.TooManyRequests(cfg.Message)
		}

		return c.Next()
	}
}

/* ───────────────── Target Resolvers (KeyFunc) ───────────────── */

func ipKey(c *fiber.Ctx) string {
	return "ip:" + c.IP()
}

func UserIDKey(c *fiber.Ctx) string {
	val := c.Locals("user_id")
	if val == nil {
		return "ip:" + c.IP()
	}

	switch v := val.(type) {
	case fmt.Stringer:
		if uid := v.String(); uid != "" {
			return "uid:" + uid
		}
	default:
		if uid := fmt.Sprintf("%v", v); uid != "" {
			return "uid:" + uid
		}
	}

	return "ip:" + c.IP()
}

/* ───────────────── Domain Specific Presets ───────────────── */

type Limiters struct {
	General fiber.Handler // Public unauthenticated endpoints (120 req/min)
	Auth    fiber.Handler // Registration, logins & validation check thresholds (10 req/5min)
	OTP     fiber.Handler // Critical transaction confirmation tokens metrics (5 req/10min)
	API     fiber.Handler // Standard authorization context limits (600 req/hour)
}

func NewLimiters(r *sharedRedis.Redis) *Limiters {
	return &Limiters{
		General: RateLimiter(r, LimitConfig{
			Max:     120,
			Window:  time.Minute,
			Message: "too many requests, please wait a moment and try again",
		}),
		Auth: RateLimiter(r, LimitConfig{
			Max:     10,
			Window:  5 * time.Minute,
			Message: "too many attempts, please wait 5 minutes and try again",
		}),
		OTP: RateLimiter(r, LimitConfig{
			Max:     5,
			Window:  10 * time.Minute,
			Message: "otp request limit reached, please wait 10 minutes",
		}),
		API: RateLimiter(r, LimitConfig{
			Max:     600,
			Window:  time.Hour,
			KeyFunc: UserIDKey,
			Message: "api rate limit exceeded, resets in one hour",
		}),
	}
}