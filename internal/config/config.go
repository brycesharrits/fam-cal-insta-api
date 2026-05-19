package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type TokenCosts struct {
	FullCalendarGeneration  int
	SingleMonthRegeneration int
	PDFExport               int
}

type Config struct {
	Port   string
	AppEnv string

	DatabaseURL string

	JWTSecret string
	JWTExpiry time.Duration

	ReplicateAPIKey       string
	ReplicateFluxModel    string
	ReplicateWebhookURL   string
	ReplicateWebhookSecret string

	S3Endpoint  string
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string

	AppleTeamID        string
	AppleKeyID         string
	ApplePrivateKeyPath string

	TokenCosts TokenCosts
}

func Load() (*Config, error) {
	jwtExpiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "720"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}

	costFull, _ := strconv.Atoi(getEnv("TOKEN_COST_FULL_GENERATION", "12"))
	costRegen, _ := strconv.Atoi(getEnv("TOKEN_COST_SINGLE_REGEN", "1"))
	costPDF, _ := strconv.Atoi(getEnv("TOKEN_COST_PDF_EXPORT", "5"))

	cfg := &Config{
		Port:   getEnv("PORT", "8080"),
		AppEnv: getEnv("APP_ENV", "development"),

		DatabaseURL: requireEnv("DATABASE_URL"),

		JWTSecret: requireEnv("JWT_SECRET"),
		JWTExpiry: time.Duration(jwtExpiryHours) * time.Hour,

		ReplicateAPIKey:        requireEnv("REPLICATE_API_KEY"),
		ReplicateFluxModel:     getEnv("REPLICATE_FLUX_MODEL", "black-forest-labs/flux-1.1-pro"),
		ReplicateWebhookURL:    getEnv("REPLICATE_WEBHOOK_URL", ""),
		ReplicateWebhookSecret: getEnv("REPLICATE_WEBHOOK_SECRET", ""),

		S3Endpoint:  getEnv("S3_ENDPOINT", ""),
		S3Bucket:    requireEnv("S3_BUCKET"),
		S3Region:    getEnv("S3_REGION", "us-east-1"),
		S3AccessKey: requireEnv("S3_ACCESS_KEY"),
		S3SecretKey: requireEnv("S3_SECRET_KEY"),

		AppleTeamID:        getEnv("APPLE_TEAM_ID", ""),
		AppleKeyID:         getEnv("APPLE_KEY_ID", ""),
		ApplePrivateKeyPath: getEnv("APPLE_PRIVATE_KEY_PATH", ""),

		TokenCosts: TokenCosts{
			FullCalendarGeneration:  costFull,
			SingleMonthRegeneration: costRegen,
			PDFExport:               costPDF,
		},
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}
