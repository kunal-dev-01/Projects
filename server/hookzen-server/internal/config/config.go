package config

import (
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	JWT      JWTConfig
	AdminJWT JWTConfig
	Security SecurityConfig
	Frontend FrontendConfig
	AWS      AWSConfig
	Google   GoogleConfig
	Github   GitHubConfig
	App      AppConfig
	R2       R2Config
	Logger   LoggerConfig
	Paddle   PaddleConfig
	MongoDB  MongoDBConfig
}

type AppConfig struct {
	AppName           string
	Env               string
	Port              int
	RequestBodyLimit  int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	TrustedProxies    []string
	AllowedOrigins    []string
	AllowedMethods    []string
	AllowedHeaders    []string
	ExposedHeaders    []string
	AllowCredentials  bool
	ReadHeaderTimeout time.Duration
	EnableMetrics     bool
	EnableHealth      bool
	TempStoragePath   string
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
	RetryAttempts   int
	RetryDelay      time.Duration
}

type RedisConfig struct {
	Host               string
	Port               int
	Password           string
	DB                 int
	MaxRetries         int
	PoolSize           int
	PoolTimeout        time.Duration
	MaxConnAge         time.Duration
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	RetryAttempts      int
	RetryDelay         time.Duration
}

type RabbitMQConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	VHost       string
	MaxRetries  int
	RetryDelay  time.Duration
	EmailQueue  string
	AvatarQueue string
	MediaQueue  string
	ChatQueue   string
}

type JWTConfig struct {
	Secret               string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}

type SecurityConfig struct {
	MaxLoginAttempts  int
	LockoutDuration   time.Duration
	RateLimitMax      int
	MaxSessions       int
	TOTPEncryptionKey string
	VaultSecretKey    string
}

type FrontendConfig struct {
	FrontendUrl           string
	OAuthSuccessURL       string
	OAuthErrorURL         string
	VerificationUrl       string
	ResetUrl              string
	AdminResetPasswordUrl string
	SecretResetUrl        string
}

type AWSConfig struct {
	AWSAccessKey string
	AWSSecretKey string
	AWSRegion    string
	AWSFromEmail string
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type PaddleConfig struct {
	Environment   string
	SellerID      string
	APIKey        string
	ClientToken   string
	WebhookSecret string
}

type MongoDBConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	DBName         string
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

type R2Config struct {
	AccountID  string
	AccessKey  string
	SecretKey  string
	BucketName string
	PublicURL  string
}

type LoggerConfig struct {
	Level              string
	EnableConsole      bool
	EnableFile         bool
	FilePath           string
	MaxSize            int
	MaxBackups         int
	MaxAge             int
	Compress           bool
	EnableSampling     bool
	SamplingInitial    int
	SamplingThereafter int
	EnableCaller       bool
	EnableStackTrace   bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", "postgres"),
			SSLMode:         getEnvAsBool("DB_SSLMODE", false),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 30),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 50),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 2*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
			ConnectTimeout:  getEnvAsDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
			QueryTimeout:    getEnvAsDuration("DB_QUERY_TIMEOUT", 30*time.Second),
			RetryAttempts:   getEnvAsInt("DB_RETRY_ATTEMPTS", 3),
			RetryDelay:      getEnvAsDuration("DB_RETRY_DELAY", 5*time.Second),
		},
		Redis: RedisConfig{
			Host:               getEnv("REDIS_HOST", "localhost"),
			Port:               getEnvAsInt("REDIS_PORT", 6379),
			Password:           getEnv("REDIS_PASSWORD", ""),
			DB:                 getEnvAsInt("REDIS_DB", 0),
			MaxRetries:         getEnvAsInt("REDIS_MAX_RETRIES", 3),
			PoolSize:           getEnvAsInt("REDIS_POOL_SIZE", 10),
			PoolTimeout:        getEnvAsDuration("REDIS_POOL_TIMEOUT", 4*time.Second),
			MaxConnAge:         getEnvAsDuration("REDIS_MAX_CONN_AGE", 0),
			IdleTimeout:        getEnvAsDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
			IdleCheckFrequency: getEnvAsDuration("REDIS_IDLE_CHECK_FREQ", 1*time.Minute),
			DialTimeout:        getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:        getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:       getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			RetryAttempts:      getEnvAsInt("REDIS_RETRY_ATTEMPTS", 3),
			RetryDelay:         getEnvAsDuration("REDIS_RETRY_DELAY", 5*time.Second),
		},
		RabbitMQ: RabbitMQConfig{
			Host:        getEnv("RABBITMQ_HOST", "localhost"),
			Port:        getEnvAsInt("RABBITMQ_PORT", 5672),
			User:        getEnv("RABBITMQ_USER", "guest"),
			Password:    getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:       getEnv("RABBITMQ_VHOST", ""),
			MaxRetries:  getEnvAsInt("RABBITMQ_MAX_RETRIES", 5),
			RetryDelay:  getEnvAsDuration("RABBITMQ_RETRY_DELAY", 5*time.Second),
			EmailQueue:  getEnv("RABBITMQ_EMAIL_QUEUE", "email_queue"),
			AvatarQueue: getEnv("RABBITMQ_AVATAR_QUEUE", "avatar_queue"),
			MediaQueue:  getEnv("RABBITMQ_MEDIA_QUEUE", "media_optimization"),
			ChatQueue:   getEnv("RABBITMQ_CHAT_QUEUE", "chat_events"),
		},
		JWT: JWTConfig{
			Secret:               getEnv("JWT_SECRET", ""),
			AccessTokenDuration:  getEnvAsDuration("ACCESS_TOKEN_DURATION", 15*time.Minute),
			RefreshTokenDuration: getEnvAsDuration("REFRESH_TOKEN_DURATION", 7*24*time.Hour),
			Issuer:               getEnv("JWT_ISSUER", "upnixo.com"),
		},
		AdminJWT: JWTConfig{
			Secret:               getEnv("ADMIN_JWT_SECRET", ""),
			AccessTokenDuration:  getEnvAsDuration("ADMIN_ACCESS_TOKEN_DURATION", 15*time.Minute),
			RefreshTokenDuration: getEnvAsDuration("ADMIN_REFRESH_TOKEN_DURATION", 7*24*time.Hour),
			Issuer:               getEnv("ADMIN_JWT_ISSUER", "upnixo.com"),
		},
		Security: SecurityConfig{
			MaxLoginAttempts:  getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:   getEnvAsDuration("LOCKOUT_DURATION", 30*time.Minute),
			RateLimitMax:      getEnvAsInt("RATE_LIMIT_MAX", 100),
			MaxSessions:       getEnvAsInt("MAX_SESSIONS", 5),
			TOTPEncryptionKey: getEnv("TOTP_ENCRYPTION_KEY", "default_totp_encryption_key"),
			VaultSecretKey:    getEnv("VAULT_SECRET_KEY", "default_32_byte_vault_secret_key_"),
		},
		Frontend: FrontendConfig{
			FrontendUrl:           getEnv("FRONTEND_URL", "http://localhost:3000"),
			OAuthSuccessURL:       getEnv("OAUTH_SUCCESS_URL", "http://localhost:3000/auth/success"),
			OAuthErrorURL:         getEnv("OAUTH_ERROR_URL", "http://localhost:3000/auth/error"),
			VerificationUrl:       getEnv("VERIFICATION_URL", "http://localhost:3000/auth/verify"),
			ResetUrl:              getEnv("RESET_URL", "http://localhost:3000/auth/reset"),
			AdminResetPasswordUrl: getEnv("ADMIN_RESET_PASSWORD_URL", "http://localhost:3000/admin/reset-password"),
			SecretResetUrl:        getEnv("SECRET_RESET_URL", "http://localhost:3000/auth/secret-reset"),
		},
		AWS: AWSConfig{
			AWSAccessKey: getEnv("AWS_ACCESS_KEY_ID", ""),
			AWSSecretKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			AWSRegion:    getEnv("AWS_REGION", "eu-central-1"),
			AWSFromEmail: getEnv("AWS_FROM_EMAIL", ""),
		},
		Google: GoogleConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
		},
		Github: GitHubConfig{
			ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
			ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GITHUB_REDIRECT_URL", "http://localhost:8080/auth/github/callback"),
		},
		Paddle: PaddleConfig{
			Environment:   getEnv("PADDLE_ENVIRONMENT", "sandbox"),
			SellerID:      getEnv("PADDLE_SELLER_ID", ""),
			APIKey:        getEnv("PADDLE_API_KEY", ""),
			ClientToken:   getEnv("PADDLE_CLIENT_TOKEN", ""),
			WebhookSecret: getEnv("PADDLE_WEBHOOK_SECRET", ""),
		},
		App: AppConfig{
			AppName:           getEnv("APP_NAME", "Upnixo"),
			Env:               getEnv("ENV", "development"),
			Port:              getEnvAsInt("PORT", 8080),
			RequestBodyLimit:  getEnvAsInt("REQUEST_BODY_LIMIT", 10*1024*1024),
			ReadTimeout:       getEnvAsDuration("READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      getEnvAsDuration("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:       getEnvAsDuration("IDLE_TIMEOUT", 1*time.Minute),
			TrustedProxies:    getEnvAsSlice("TRUSTED_PROXIES", []string{"localhost", "[IP_ADDRESS]"}),
			AllowedOrigins:    getEnvAsSlice("ALLOWED_ORIGINS", []string{"*"}),
			AllowedMethods:    getEnvAsSlice("ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders:    getEnvAsSlice("ALLOWED_HEADERS", []string{"Authorization", "Content-Type"}),
			ExposedHeaders:    getEnvAsSlice("EXPOSED_HEADERS", []string{"Authorization"}),
			AllowCredentials:  getEnvAsBool("ALLOW_CREDENTIALS", true),
			ReadHeaderTimeout: getEnvAsDuration("READ_HEADER_TIMEOUT", 10*time.Second),
			EnableMetrics:     getEnvAsBool("ENABLE_METRICS", true),
			EnableHealth:      getEnvAsBool("ENABLE_HEALTH", true),
			TempStoragePath:   getEnv("TEMP_STORAGE_PATH", "./storage/tmp"),
		},
		Logger: LoggerConfig{
			Level:              getEnv("LOG_LEVEL", "info"),
			EnableConsole:      getEnvAsBool("LOG_ENABLE_CONSOLE", true),
			EnableFile:         getEnvAsBool("LOG_ENABLE_FILE", false),
			FilePath:           getEnv("LOG_FILE_PATH", "/var/log/app/app.log"),
			MaxSize:            getEnvAsInt("LOG_MAX_SIZE", 100),
			MaxBackups:         getEnvAsInt("LOG_MAX_BACKUPS", 10),
			MaxAge:             getEnvAsInt("LOG_MAX_AGE", 30),
			Compress:           getEnvAsBool("LOG_COMPRESS", true),
			EnableSampling:     getEnvAsBool("LOG_ENABLE_SAMPLING", false),
			SamplingInitial:    getEnvAsInt("LOG_SAMPLING_INITIAL", 10),
			SamplingThereafter: getEnvAsInt("LOG_SAMPLING_THEREAFTER", 100),
			EnableCaller:       getEnvAsBool("LOG_ENABLE_CALLER", false),
			EnableStackTrace:   getEnvAsBool("LOG_ENABLE_STACKTRACE", true),
		},
		R2: R2Config{
			AccountID:  getEnv("R2_ACCOUNT_ID", ""),
			AccessKey:  getEnv("R2_ACCESS_KEY", ""),
			SecretKey:  getEnv("R2_SECRET_KEY", ""),
			BucketName: getEnv("R2_BUCKET_NAME", ""),
			PublicURL:  getEnv("R2_PUBLIC_URL", "http://upraizo.com"),
		},
		MongoDB: MongoDBConfig{
			Host:           getEnv("MONGO_HOST", "localhost"),
			Port:           getEnvAsInt("MONGO_PORT", 27017),
			User:           getEnv("MONGO_USER", "admin"),
			Password:       getEnv("MONGO_PASSWORD", "password"),
			DBName:         getEnv("MONGO_DB", "upnixo_chat"),
			ConnectTimeout: getEnvAsDuration("MONGO_CONNECT_TIMEOUT", 10*time.Second),
			QueryTimeout:   getEnvAsDuration("MONGO_QUERY_TIMEOUT", 30*time.Second),
			RetryAttempts:  getEnvAsInt("MONGO_RETRY_ATTEMPTS", 3),
			RetryDelay:     getEnvAsDuration("MONGO_RETRY_DELAY", 5*time.Second),
		},
	}

	if cfg.Database.Password == "" && !cfg.App.IsDevelopment() {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	if len(cfg.JWT.Secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	if cfg.App.IsProduction() && cfg.Redis.Password == "" {
		return nil, fmt.Errorf("REDIS_PASSWORD is required in production")
	}

	if cfg.AWS.AWSAccessKey == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID is required")
	}

	if cfg.AWS.AWSSecretKey == "" {
		return nil, fmt.Errorf("AWS_SECRET_ACCESS_KEY is required")
	}

	if cfg.Google.ClientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}

	if cfg.Paddle.APIKey == "" {
		return nil, fmt.Errorf("PADDLE_API_KEY is required")
	}

	if cfg.App.IsProduction() && cfg.Paddle.WebhookSecret == "" {
		return nil, fmt.Errorf("PADDLE_WEBHOOK_SECRET is required in production")
	}

	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	if !slices.Contains(validLevels, cfg.Logger.Level) {
		return nil, fmt.Errorf("LOG_LEVEL must be one of: %v", validLevels)
	}

	return cfg, nil
}

/* ───────────────── Helper Functions ───────────────── */

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}

func getEnvAsSlice(key string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return fallback
}

/* ───────────────── DSN Builders ───────────────── */

func (c *DatabaseConfig) GetDSN() string {
	sslModeStr := "disable"
	if c.SSLMode {
		sslModeStr = "require"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		c.Host, c.Port, c.User, url.QueryEscape(c.Password), c.DBName, sslModeStr, int(c.ConnectTimeout.Seconds()))
}

func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *RabbitMQConfig) GetURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		url.QueryEscape(c.User), url.QueryEscape(c.Password), c.Host, c.Port, c.VHost)
}

func (c *MongoDBConfig) GetURI() string {
	if c.User != "" && c.Password != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=admin",
			url.QueryEscape(c.User), url.QueryEscape(c.Password), c.Host, c.Port, c.DBName)
	}
	return fmt.Sprintf("mongodb://%s:%d/%s", c.Host, c.Port, c.DBName)
}

/* ───────────────── Environment-Specific Helpers ───────────────── */

func (c AppConfig) IsDevelopment() bool {
	return c.Env == "development" || c.Env == ""
}

func (c AppConfig) IsProduction() bool {
	return c.Env == "production"
}

func (c AppConfig) IsStaging() bool {
	return c.Env == "staging"
}