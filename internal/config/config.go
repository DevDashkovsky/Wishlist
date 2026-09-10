package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	JWTExpiry     time.Duration
	MigrationsDir string
}

func Load() (*Config, error) {
	port := getEnv("PORT", "8080")
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("PORT must be an integer between 1 and 65535")
	}

	dsn, err := requireEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	jwtSecret, err := requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	expiryRaw := getEnv("JWT_EXPIRY_MINUTES", "60")
	expiryMinutes, err := strconv.Atoi(expiryRaw)
	if err != nil {
		return nil, fmt.Errorf("JWT_EXPIRY_MINUTES must be an integer number of minutes, got %q: %w", expiryRaw, err)
	}
	if expiryMinutes <= 0 || int64(expiryMinutes) > math.MaxInt64/int64(time.Minute) {
		return nil, fmt.Errorf("JWT_EXPIRY_MINUTES must be positive and fit in time.Duration, got %d", expiryMinutes)
	}

	return &Config{
		Port:          port,
		DatabaseURL:   dsn,
		JWTSecret:     jwtSecret,
		JWTExpiry:     time.Duration(expiryMinutes) * time.Minute,
		MigrationsDir: getEnv("MIGRATIONS_DIR", "./migrations"),
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
