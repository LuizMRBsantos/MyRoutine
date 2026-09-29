package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
	JWTSecret            string
	JWTExpiryHours       string
	JWTRefreshExpiryDays string

	// Claude AI
	AnthropicAPIKey string
	AnthropicModel  string

	// CORS
	CORSAllowedOrigins string

	// AdminEmails can register without an invite and are promoted to admin
	// at startup. Normalized (trimmed, lowercased). Registration is
	// invite-only for everyone else.
	AdminEmails []string

	// TrustedIPHeader names the header carrying the real client IP, set by
	// the proxy in front: X-Real-IP (nginx, local) or CloudFront-Viewer-Address
	// (AWS). Only safe when nothing but that proxy can reach the API.
	TrustedIPHeader string

	// AuthRateLimitPerMinute caps auth requests per client IP. 0 disables it
	// (tests only — Load() refuses 0).
	AuthRateLimitPerMinute int
}

// IsAdminEmail reports whether email (any case/whitespace) is in AdminEmails.
func (c *Config) IsAdminEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, admin := range c.AdminEmails {
		if admin == email {
			return true
		}
	}
	return false
}

// parseEmailList splits a comma-separated list, dropping blanks and
// normalizing each entry.
func parseEmailList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if e := strings.ToLower(strings.TrimSpace(part)); e != "" {
			out = append(out, e)
		}
	}
	return out
}

// Load reads configuration from environment variables.
// In development, it also loads from a .env file if present.
func Load() (*Config, error) {
	// Load .env in non-production environments.
	// Tries the working directory first, then the repo root (when running
	// from backend/). The file is optional — absence is not an error.
	if os.Getenv("APP_ENV") != "production" {
		if err := godotenv.Load(".env"); err != nil {
			_ = godotenv.Load("../.env")
		}
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

		AdminEmails: parseEmailList(os.Getenv("ADMIN_EMAILS")),

		TrustedIPHeader: getEnv("TRUSTED_IP_HEADER", "X-Real-IP"),
	}

	limit, err := strconv.Atoi(getEnv("AUTH_RATE_LIMIT_PER_MINUTE", "10"))
	if err != nil || limit <= 0 {
		return nil, fmt.Errorf("AUTH_RATE_LIMIT_PER_MINUTE must be a positive integer")
	}
	cfg.AuthRateLimitPerMinute = limit

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
	if err := requirePositiveInt("JWT_EXPIRY_HOURS", c.JWTExpiryHours); err != nil {
		return err
	}
	if err := requirePositiveInt("JWT_REFRESH_EXPIRY_DAYS", c.JWTRefreshExpiryDays); err != nil {
		return err
	}
	return nil
}

// requirePositiveInt fails fast when an env var is not an integer > 0. A bad
// value would otherwise parse as 0 and every token would be born expired.
func requirePositiveInt(key, value string) error {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fmt.Errorf("%s must be a positive integer, got %q", key, value)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
