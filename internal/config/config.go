package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
// Add new fields here as the app grows — never call os.Getenv outside this package.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Email    EmailConfig
	Storage  StorageConfig
}

type AppConfig struct {
	Port                         string
	Env                          string // "development" | "production"
	OpenAIAPIKey                 string // OpenAI API key for Moderation API (optional)
	PerspectiveAPIKey            string // Google Perspective API key (optional, legacy)
	CloudVisionAPIKey            string // Google Cloud Vision API key (optional, legacy)
	ScholarshipsRemindersEnabled bool   // Phase 3 — daily scholarship deadline reminder worker
}

type DatabaseConfig struct {
	URL string
}

type AuthConfig struct {
	JWTSecret          string
	JWTRefreshSecret   string
	JWTExpirationHours int
}

type EmailConfig struct {
	ResendAPIKey string
	FromAddress  string
	FromName     string
}

type StorageConfig struct {
	AccountID         string
	AccessKey         string
	SecretKey         string
	Bucket            string
	PublicDomain      string
	VideoMaxSizeMB    int // Max size in MB for uploaded videos (default 200)
	VideoMaxDurationS int // Max duration in seconds for uploaded videos (default 30)
}

// Load reads all required environment variables and returns a validated Config.
// Returns an error listing every missing variable so operators can fix all at once.
func Load() (*Config, error) {
	var missing []string

	required := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	optional := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}

	optionalInt := func(key string, fallback int) int {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return fallback
	}

	optionalBool := func(key string, fallback bool) bool {
		if v := os.Getenv(key); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}
		return fallback
	}

	cfg := &Config{
		App: AppConfig{
			Port:                         optional("PORT", "8080"),
			Env:                          optional("APP_ENV", "development"),
			OpenAIAPIKey:                 optional("OPENAI_API_KEY", ""),
			PerspectiveAPIKey:            optional("PERSPECTIVE_API_KEY", ""),
			CloudVisionAPIKey:            optional("CLOUD_VISION_API_KEY", ""),
			ScholarshipsRemindersEnabled: optionalBool("SCHOLARSHIPS_REMINDERS_ENABLED", false),
		},
		Database: DatabaseConfig{
			URL: required("DATABASE_URL"),
		},
		Auth: AuthConfig{
			JWTSecret:          required("JWT_SECRET"),
			JWTExpirationHours: optionalInt("JWT_EXPIRATION_HOURS", 24),
			JWTRefreshSecret:   required("JWT_REFRESH_SECRET"),
		},
		Email: EmailConfig{
			ResendAPIKey: required("RESEND_API_KEY"),
			FromAddress:  optional("EMAIL_FROM_ADDRESS", "mail.mldsoftware.com"),
			FromName:     optional("EMAIL_FROM_NAME", "Hilos"),
		},
		Storage: StorageConfig{
			AccountID:         required("ACCOUNT_ID"),
			AccessKey:         required("ACCESS_KEY_ID"),
			SecretKey:         required("SECRET_ACCESS_KEY"),
			Bucket:            required("BUCKET_NAME"),
			PublicDomain:      required("PUBLIC_DOMAIN"),
			VideoMaxSizeMB:    optionalInt("VIDEO_MAX_SIZE_MB", 200),
			VideoMaxDurationS: optionalInt("VIDEO_MAX_DURATION_S", 30),
		},
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}
