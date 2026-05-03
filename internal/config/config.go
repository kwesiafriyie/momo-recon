package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	MoMoBaseURL         string
	MoMoSubscriptionKey string
	MoMoAPIUserID       string
	MoMoAPIKey          string
	MoMoCallbackURL     string
	MoMoTargetEnv       string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         requireEnv("DATABASE_URL"),
		MoMoBaseURL:         getEnv("MOMO_BASE_URL", "https://sandbox.momodeveloper.mtn.com"),
		MoMoSubscriptionKey: requireEnv("MOMO_SUBSCRIPTION_KEY"),
		MoMoAPIUserID:       requireEnv("MOMO_API_USER_ID"),
		MoMoAPIKey:          requireEnv("MOMO_API_KEY"),
		MoMoCallbackURL:     requireEnv("MOMO_CALLBACK_URL"),
		MoMoTargetEnv:       getEnv("MOMO_TARGET_ENV", "sandbox"),
	}

	return cfg, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
