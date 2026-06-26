package config

import (
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "secret")
}

func TestLoad_Defaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.JWTExpiry != 60*time.Minute {
		t.Errorf("JWTExpiry = %v, want 60m", cfg.JWTExpiry)
	}
	if cfg.MigrationsDir != "./migrations" {
		t.Errorf("MigrationsDir = %q, want ./migrations", cfg.MigrationsDir)
	}
}

func TestLoad_InvalidJWTExpiry(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("JWT_EXPIRY_MINUTES", "60m") // a typo that used to be silently swallowed

	if _, err := Load(); err == nil {
		t.Fatal("expected error for non-integer JWT_EXPIRY_MINUTES, got nil")
	}
}

func TestLoad_NonPositiveJWTExpiry(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("JWT_EXPIRY_MINUTES", "0")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for zero JWT_EXPIRY_MINUTES, got nil")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "") // requireEnv treats empty as unset
	t.Setenv("JWT_SECRET", "secret")

	if _, err := Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL is missing, got nil")
	}
}
