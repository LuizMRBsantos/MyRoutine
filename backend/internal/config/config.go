package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
// All secrets come from env vars — never hardcoded.
type Config struct {
	// App
	AppEnv  string
	AppPort string
	AppName string

	// Database
	DatabaseURL string
	DBHost      string
	DBPort      string
	DBUser      string
	DBName      string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// JWT
	JWTSecret             string
	JWTExpiryHours        string
	JWTRefreshExpiryDays  string

	// Claude AI
	AnthropicAPIKey string
	AnthropicModel  string

	// CORS
	CORSAllowedOrigins string
}

// Load reads configuration from environment variables.
// In development, it also loads from a .env file if present.
func Load() (*Config, error) {
	// Load .env in non-production environments
	if os.Getenv("APP_ENV") != "production" {
		// Ignore error — .env is optional in CI/CD
		_ = godotenv.Load("../.env")
	}

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),
		AppName: getEnv("APP_NAME", "myroutine"),

		DatabaseURL: getEnv("DATABASE_URL", ""),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "myroutine"),
		DBName:      getEnv("DB_NAME", "myroutine_db"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		JWTSecret:            getEnv("JWT_SECRET", ""),
		JWTExpiryHours:       getEnv("JWT_EXPIRY_HOURS", "24"),
		JWTRefreshExpiryDays: getEnv("JWT_REFRESH_EXPIRY_DAYS", "30"),

		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicModel:  getEnv("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022"),

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks required fields are present.
func (c *Config) validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
