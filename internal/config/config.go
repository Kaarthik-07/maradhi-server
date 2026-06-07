package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port            string
	Env             string
	DatabaseURL     string
	JWTSecret       string
	AdminSecret     string
	AllowedOrigins  []string
	AnthropicAPIKey string
}

func (c *Config) IsDev() bool { return c.Env == "development" }

func Load() (*Config, error) {
	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		Env:             getEnv("ENV", "development"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AdminSecret:     os.Getenv("ADMIN_SECRET"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.AdminSecret == "" {
		return nil, fmt.Errorf("ADMIN_SECRET is required")
	}
	raw := getEnv("ALLOWED_ORIGINS", "http://localhost:3000")
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
