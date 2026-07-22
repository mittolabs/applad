package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might be set
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_DSN")
	os.Unsetenv("REDIS_ADDR")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("STORAGE_PATH")
	os.Unsetenv("APP_ENV")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("expected default env 'development', got %s", cfg.AppEnv)
	}
	if cfg.JWTSecret != "change-me-in-production" {
		t.Fatalf("expected default JWT secret, got %s", cfg.JWTSecret)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("APP_ENV", "production")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("APP_ENV")
	}()

	cfg := Load()

	if cfg.Port != "9090" {
		t.Fatalf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.AppEnv != "production" {
		t.Fatalf("expected env 'production', got %s", cfg.AppEnv)
	}
}

func TestGetEnv_Fallback(t *testing.T) {
	os.Unsetenv("TEST_NONEXISTENT")
	val := getEnv("TEST_NONEXISTENT", "default")
	if val != "default" {
		t.Fatalf("expected 'default', got %s", val)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("TEST_SET_VAR", "custom")
	defer os.Unsetenv("TEST_SET_VAR")

	val := getEnv("TEST_SET_VAR", "default")
	if val != "custom" {
		t.Fatalf("expected 'custom', got %s", val)
	}
}
