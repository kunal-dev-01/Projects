package postgresql

import (
	"context"
	"fmt"
	"time"

	"shared/logging"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DSN             string
	Host            string
	Port            int
	DBName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	ConnectTimeout  time.Duration
	RetryAttempts   int
	RetryDelay      time.Duration
}

type Postgres struct {
	Pool *pgxpool.Pool
	cfg  Config
}

func New(cfg Config) *Postgres {
	return &Postgres{cfg: cfg}
}

func (p *Postgres) Connect(ctx context.Context) error {
	poolConfig, err := pgxpool.ParseConfig(p.cfg.DSN)
	if err != nil {
		logging.Error("failed to parse database DSN", err, nil)
		return fmt.Errorf("parse dsn: %w", err)
	}

	poolConfig.MaxConns = int32(p.cfg.MaxOpenConns)
	poolConfig.MinConns = int32(p.cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = p.cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = p.cfg.ConnMaxIdleTime
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := p.connectWithRetry(ctx, poolConfig)
	if err != nil {
		logging.Error("failed to connect to database after maximum retries", err, nil)
		return fmt.Errorf("database connection failed: %w", err)
	}

	p.Pool = pool

	logging.Info("postgresql connected", map[string]any{
		"host":      p.cfg.Host,
		"port":      p.cfg.Port,
		"db":        p.cfg.DBName,
		"max_conns": p.cfg.MaxOpenConns,
		"min_conns": p.cfg.MaxIdleConns,
		"driver":    "pgx/v5",
	})

	return nil
}

func (p *Postgres) Close() {
	if p.Pool == nil {
		return
	}
	p.Pool.Close()
	logging.Info("database connection pool closed", nil)
}

func (p *Postgres) connectWithRetry(ctx context.Context, poolConfig *pgxpool.Config) (*pgxpool.Pool, error) {
	var (
		pool *pgxpool.Pool
		err  error
	)

	for attempt := 1; attempt <= p.cfg.RetryAttempts; attempt++ {
		pool, err = p.tryConnect(ctx, poolConfig)
		if err == nil {
			return pool, nil
		}

		logging.Warn(fmt.Sprintf("database connection attempt %d/%d failed", attempt, p.cfg.RetryAttempts), map[string]any{
			"error": err.Error(),
		})

		if attempt < p.cfg.RetryAttempts {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("database: connect cancelled: %w", ctx.Err())
			case <-time.After(p.cfg.RetryDelay):
			}
		}
	}

	return nil, err
}

func (p *Postgres) tryConnect(ctx context.Context, poolConfig *pgxpool.Config) (*pgxpool.Pool, error) {
	connectCtx, cancel := context.WithTimeout(ctx, p.cfg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}