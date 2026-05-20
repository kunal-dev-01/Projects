package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	UserIDKey    ContextKey = "user_id"
)

type Config struct {
	Level              string
	Env                string
	EnableConsole      bool
	EnableFile         bool
	FilePath           string
	MaxSize            int
	MaxBackups         int
	MaxAge             int
	Compress           bool
	EnableSampling     bool
	SamplingThereafter int
	EnableCaller       bool
	EnableStackTrace   bool
}

type Logger struct {
	log              zerolog.Logger
	enableStackTrace bool
}

var (
	globalLogger *Logger
	initOnce     sync.Once
)

func Init(cfg Config) {
	initOnce.Do(func() {
		zerolog.TimeFieldFormat = time.RFC3339

		level, err := zerolog.ParseLevel(cfg.Level)
		if err != nil {
			level = zerolog.InfoLevel
		}

		writers := buildWriters(cfg)
		output := zerolog.MultiLevelWriter(writers...)

		zCtx := zerolog.New(output).
			Level(level).
			With().
			Timestamp().
			Str("env", cfg.Env)

		if cfg.EnableCaller {
			zCtx = zCtx.Caller()
		}

		zl := zCtx.Logger()

		if cfg.EnableSampling && cfg.SamplingThereafter > 0 {
			zl = zl.Sample(&zerolog.BasicSampler{
				N: uint32(cfg.SamplingThereafter),
			})
		}

		globalLogger = &Logger{
			log:              zl,
			enableStackTrace: cfg.EnableStackTrace,
		}
		zerolog.DefaultContextLogger = &globalLogger.log
	})
}

func buildWriters(cfg Config) []io.Writer {
	var writers []io.Writer

	if cfg.EnableConsole {
		if cfg.Env == "development" || cfg.Env == "" {
			writers = append(writers, zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC3339,
			})
		} else {
			writers = append(writers, os.Stdout)
		}
	}

	if cfg.EnableFile && cfg.FilePath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to create log directory: %v\n", err)
		} else {
			writers = append(writers, &lumberjack.Logger{
				Filename:   cfg.FilePath,
				MaxSize:    cfg.MaxSize,
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAge,
				Compress:   cfg.Compress,
			})
		}
	}

	return writers
}

func Get() *Logger {
	if globalLogger == nil {
		zl := zerolog.New(os.Stdout).With().Timestamp().Logger()
		return &Logger{log: zl, enableStackTrace: true}
	}
	return globalLogger
}

func (l *Logger) Raw() *zerolog.Logger {
	return &l.log
}

/* ───────────────── Fiber Middlewares ───────────────── */

func (l *Logger) HTTP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		requestID, ok := c.Locals("requestid").(string)
		if !ok || requestID == "" {
			requestID = uuid.New().String()
			c.Locals("requestid", requestID)
		}

		err := c.Next()

		status := c.Response().StatusCode()
		if err != nil {
			if e, ok := err.(*fiber.Error); ok && status == fiber.StatusOK {
				status = e.Code
			}
		}

		event := l.levelFor(status)().
			Str("type", "http").
			Str("request_id", requestID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", status).
			Dur("duration_ms", time.Since(start)).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent"))

		if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
			event = event.Str("user_id", userID)
		}

		if err != nil {
			event = event.Err(err)
		}

		event.Msg("http request")
		return err
	}
}

func (l *Logger) Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			r := recover()
			if r == nil {
				return
			}

			requestID, _ := c.Locals("requestid").(string)

			event := l.log.Error().
				Str("type", "panic").
				Str("request_id", requestID).
				Str("method", c.Method()).
				Str("path", c.Path()).
				Interface("panic", r)

			if l.enableStackTrace {
				event = event.Stack()
			}

			event.Msg("panic recovered")

			_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "something went wrong",
				"ref_id":  requestID,
			})
		}()

		return c.Next()
	}
}

func (l *Logger) levelFor(status int) func() *zerolog.Event {
	switch {
	case status >= 500:
		return l.log.Error
	case status >= 400:
		return l.log.Warn
	default:
		return l.log.Info
	}
}

/* ───────────────── Context Helpers & Wrappers ───────────────── */

func (l *Logger) WithFiberCtx(c *fiber.Ctx) *zerolog.Logger {
	zl := l.log
	if reqID, ok := c.Locals("requestid").(string); ok && reqID != "" {
		zl = zl.With().Str("request_id", reqID).Logger()
	}
	if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
		zl = zl.With().Str("user_id", userID).Logger()
	}
	return &zl
}

func (l *Logger) WithCtx(ctx context.Context) *zerolog.Logger {
	zl := l.log
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		zl = zl.With().Str("request_id", reqID).Logger()
	}
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		zl = zl.With().Str("user_id", userID).Logger()
	}
	return &zl
}

func (l *Logger) Info(msg string, fields map[string]any)  { l.log.Info().Fields(fields).Msg(msg) }
func (l *Logger) Debug(msg string, fields map[string]any) { l.log.Debug().Fields(fields).Msg(msg) }
func (l *Logger) Warn(msg string, fields map[string]any)  { l.log.Warn().Fields(fields).Msg(msg) }
func (l *Logger) Error(msg string, err error, fields map[string]any) {
	l.log.Error().Err(err).Fields(fields).Msg(msg)
}
func (l *Logger) Fatal(msg string, err error, fields map[string]any) {
	l.log.Fatal().Err(err).Fields(fields).Msg(msg)
}

func Info(msg string, fields map[string]any)             { Get().Info(msg, fields) }
func Debug(msg string, fields map[string]any)            { Get().Debug(msg, fields) }
func Warn(msg string, fields map[string]any)             { Get().Warn(msg, fields) }
func Error(msg string, err error, fields map[string]any) { Get().Error(msg, err, fields) }
func Fatal(msg string, err error, fields map[string]any) { Get().Fatal(msg, err, fields) }