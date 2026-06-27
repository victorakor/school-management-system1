package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	Port        string `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`
	AppBaseURL  string `mapstructure:"APP_BASE_URL"`

	// Database
	DatabaseURL string `mapstructure:"DATABASE_URL"`

	// Redis
	RedisURL string `mapstructure:"REDIS_URL"`

	// JWT
	JWTSecret        string        `mapstructure:"JWT_SECRET"`
	JWTRefreshSecret string        `mapstructure:"JWT_REFRESH_SECRET"`
	JWTAccessTTL     time.Duration `mapstructure:"JWT_ACCESS_TTL"`
	JWTRefreshTTL    time.Duration `mapstructure:"JWT_REFRESH_TTL"`

	// CSRF
	CSRFSecret string `mapstructure:"CSRF_SECRET"`

	// Cloudinary
	CloudinaryCloudName    string `mapstructure:"CLOUDINARY_CLOUD_NAME"`
	CloudinaryAPIKey       string `mapstructure:"CLOUDINARY_API_KEY"`
	CloudinaryAPISecret    string `mapstructure:"CLOUDINARY_API_SECRET"`
	CloudinaryUploadPreset string `mapstructure:"CLOUDINARY_UPLOAD_PRESET"`

	// Email (Resend)
	ResendAPIKey string `mapstructure:"RESEND_API_KEY"`

	// Chromium
	ChromiumPath string `mapstructure:"CHROMIUM_PATH"`

	// Rate Limiting
	RateLimitRequests int    `mapstructure:"RATE_LIMIT_REQUESTS"`
	RateLimitWindow   string `mapstructure:"RATE_LIMIT_WINDOW"`

	// Sentry
	SentryDSN string `mapstructure:"SENTRY_DSN"`
}

var cfg *Config

// Load reads configuration from environment variables and optional .env file.
func Load() *Config {
	v := viper.New()

	// Read from .env file if present (development only)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig() // Ignore error — .env is optional in production

	// Bind all environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults
	v.SetDefault("PORT", "8080")
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("APP_BASE_URL", "http://localhost:8080")
	v.SetDefault("JWT_ACCESS_TTL", 15*time.Minute)
	v.SetDefault("JWT_REFRESH_TTL", 7*24*time.Hour)
	v.SetDefault("RATE_LIMIT_REQUESTS", 5)
	v.SetDefault("RATE_LIMIT_WINDOW", "15m")
	v.SetDefault("CHROMIUM_PATH", "/usr/bin/chromium")

	cfg = &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		log.Fatalf("config: failed to unmarshal: %v", err)
	}

	// Parse durations manually if set as strings
	if cfg.JWTAccessTTL == 0 {
		cfg.JWTAccessTTL = 15 * time.Minute
	}
	if cfg.JWTRefreshTTL == 0 {
		cfg.JWTRefreshTTL = 7 * 24 * time.Hour
	}

	return cfg
}

// Get returns the loaded config (panics if Load() was not called first).
func Get() *Config {
	if cfg == nil {
		log.Fatal("config: Get() called before Load()")
	}
	return cfg
}

// IsDevelopment returns true when running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true when running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
