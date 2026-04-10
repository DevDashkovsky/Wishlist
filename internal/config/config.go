package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
}

func Load() (*Config, error) {
	port := getEnv("PORT", "8080")

	dsn, err := requireEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	jwtSecret, err := requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	expiryMinutes, _ := strconv.Atoi(getEnv("JWT_EXPIRY_MINUTES", "60"))

	return &Config{
		Port:        port,
		DatabaseURL: dsn,
		JWTSecret:   jwtSecret,
		JWTExpiry:   time.Duration(expiryMinutes) * time.Minute,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("environment variable %s is required", key)
	}
	return v, nil
}
