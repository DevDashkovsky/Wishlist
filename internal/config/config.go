package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	JWTExpiry        time.Duration
	RequestTimeout   time.Duration
	MigrationTimeout time.Duration
	MigrationsDir    string
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
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}

	expiryRaw := getEnv("JWT_EXPIRY_MINUTES", "60")
	expiryMinutes, err := strconv.Atoi(expiryRaw)
	if err != nil {
		return nil, fmt.Errorf("JWT_EXPIRY_MINUTES must be an integer number of minutes, got %q: %w", expiryRaw, err)
	}
	if expiryMinutes <= 0 || int64(expiryMinutes) > math.MaxInt64/int64(time.Minute) {
		return nil, fmt.Errorf("JWT_EXPIRY_MINUTES must be positive and fit in time.Duration, got %d", expiryMinutes)
	}
	requestTimeout, err := positiveSeconds("REQUEST_TIMEOUT_SECONDS", "10")
	if err != nil {
		return nil, err
	}
	if requestTimeout > 5*time.Minute {
		return nil, fmt.Errorf("REQUEST_TIMEOUT_SECONDS must not exceed 300")
	}
	migrationTimeout, err := positiveSeconds("MIGRATION_TIMEOUT_SECONDS", "60")
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:             port,
		DatabaseURL:      dsn,
		JWTSecret:        jwtSecret,
		JWTExpiry:        time.Duration(expiryMinutes) * time.Minute,
		RequestTimeout:   requestTimeout,
		MigrationTimeout: migrationTimeout,
		MigrationsDir:    getEnv("MIGRATIONS_DIR", "./migrations"),
	}, nil
}

func positiveSeconds(key, fallback string) (time.Duration, error) {
	raw := getEnv(key, fallback)
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds <= 0 || seconds > math.MaxInt64/int64(time.Second) {
		return 0, fmt.Errorf("%s must be a positive integer that fits in time.Duration", key)
	}
	return time.Duration(seconds) * time.Second, nil
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
