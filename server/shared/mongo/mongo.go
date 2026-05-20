package mongo

import (
	"context"
	"fmt"
	"time"

	"shared/logging"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Config struct {
	URI            string
	DBName         string
	Host           string
	Port           int
	ConnectTimeout time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
	cfg      Config
}

func New(cfg Config) *MongoClient {
	return &MongoClient{cfg: cfg}
}

func (m *MongoClient) Connect(ctx context.Context) error {
	clientOptions := options.Client().ApplyURI(m.cfg.URI)

	client, err := m.connectMongoWithRetry(ctx, clientOptions)
	if err != nil {
		logging.Error("failed to connect to mongodb after maximum retries", err, nil)
		return fmt.Errorf("mongodb connection failed: %w", err)
	}

	m.Client = client
	m.Database = client.Database(m.cfg.DBName)

	logging.Info("mongodb connected", map[string]any{
		"host": m.cfg.Host,
		"port": m.cfg.Port,
		"db":   m.cfg.DBName,
	})

	return nil
}

func (m *MongoClient) Close(ctx context.Context) {
	if m.Client == nil {
		return
	}
	if err := m.Client.Disconnect(ctx); err != nil {
		logging.Error("failed to close mongodb connection", err, nil)
	}
	logging.Info("mongodb connection closed", nil)
}

func (m *MongoClient) connectMongoWithRetry(ctx context.Context, clientOptions *options.ClientOptions) (*mongo.Client, error) {
	var (
		client *mongo.Client
		err    error
	)

	for attempt := 1; attempt <= m.cfg.RetryAttempts; attempt++ {
		client, err = m.tryConnectMongo(ctx, clientOptions)
		if err == nil {
			return client, nil
		}

		logging.Warn(fmt.Sprintf("mongodb connection attempt %d/%d failed", attempt, m.cfg.RetryAttempts), map[string]any{
			"error": err.Error(),
		})

		if attempt < m.cfg.RetryAttempts {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("mongodb: connect cancelled: %w", ctx.Err())
			case <-time.After(m.cfg.RetryDelay):
			}
		}
	}

	return nil, err
}

func (m *MongoClient) tryConnectMongo(ctx context.Context, clientOptions *options.ClientOptions) (*mongo.Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, m.cfg.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	if err = client.Ping(connectCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(connectCtx)
		return nil, err
	}

	return client, nil
}