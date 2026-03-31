package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoaderLoadReadsEnvFile(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("PLAYER_STATS_API_HTTP_ADDR", "")
	t.Setenv("PLAYER_STATS_API_METRICS_ADDR", "")
	t.Setenv("PLAYER_STATS_API_READ_TIMEOUT", "")
	t.Setenv("PLAYER_STATS_API_WRITE_TIMEOUT", "")
	t.Setenv("PLAYER_STATS_API_IDLE_TIMEOUT", "")
	t.Setenv("PLAYER_STATS_API_SHUTDOWN_TIMEOUT", "")
	t.Setenv("PLAYER_STATS_API_DEFAULT_LIMIT", "")
	t.Setenv("PLAYER_STATS_API_MAX_LIMIT", "")
	t.Setenv("PLAYER_STATS_API_MINIMUM_MATCHES", "")

	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envPath, []byte(`
DATABASE_URL=postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable
APP_ENV=local
LOG_LEVEL=debug
PLAYER_STATS_API_HTTP_ADDR=:8088
PLAYER_STATS_API_METRICS_ADDR=:9192
PLAYER_STATS_API_READ_TIMEOUT=3s
PLAYER_STATS_API_WRITE_TIMEOUT=7s
PLAYER_STATS_API_IDLE_TIMEOUT=45s
PLAYER_STATS_API_SHUTDOWN_TIMEOUT=9s
PLAYER_STATS_API_DEFAULT_LIMIT=25
PLAYER_STATS_API_MAX_LIMIT=200
PLAYER_STATS_API_MINIMUM_MATCHES=12
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, err := NewLoader().Load(envPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.AppEnv != "local" {
		t.Fatalf("AppEnv = %q, want local", cfg.AppEnv)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.HTTPAddr != ":8088" {
		t.Fatalf("HTTPAddr = %q, want :8088", cfg.HTTPAddr)
	}
	if cfg.MetricsAddr != ":9192" {
		t.Fatalf("MetricsAddr = %q, want :9192", cfg.MetricsAddr)
	}
	if cfg.ReadTimeout != 3*time.Second {
		t.Fatalf("ReadTimeout = %s, want 3s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 7*time.Second {
		t.Fatalf("WriteTimeout = %s, want 7s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 45*time.Second {
		t.Fatalf("IdleTimeout = %s, want 45s", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 9*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 9s", cfg.ShutdownTimeout)
	}
	if cfg.DefaultLimit != 25 {
		t.Fatalf("DefaultLimit = %d, want 25", cfg.DefaultLimit)
	}
	if cfg.MaxLimit != 200 {
		t.Fatalf("MaxLimit = %d, want 200", cfg.MaxLimit)
	}
	if cfg.MinimumMatches != 12 {
		t.Fatalf("MinimumMatches = %d, want 12", cfg.MinimumMatches)
	}
}

func TestLoaderLoadKeepsExistingEnvironmentValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://from-os")
	t.Setenv("PLAYER_STATS_API_HTTP_ADDR", ":18080")
	t.Setenv("PLAYER_STATS_API_DEFAULT_LIMIT", "75")

	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envPath, []byte(`
DATABASE_URL=postgres://from-file
PLAYER_STATS_API_HTTP_ADDR=:8080
PLAYER_STATS_API_DEFAULT_LIMIT=25
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, err := NewLoader().Load(envPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://from-os" {
		t.Fatalf("DatabaseURL = %q, want postgres://from-os", cfg.DatabaseURL)
	}
	if cfg.HTTPAddr != ":18080" {
		t.Fatalf("HTTPAddr = %q, want :18080", cfg.HTTPAddr)
	}
	if cfg.DefaultLimit != 75 {
		t.Fatalf("DefaultLimit = %d, want 75", cfg.DefaultLimit)
	}
}

func TestLoaderLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable")
	t.Setenv("PLAYER_STATS_API_READ_TIMEOUT", "soon")

	_, err := NewLoader().Load("")
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse player_stats_api_read_timeout") {
		t.Fatalf("Load() error = %v, want duration parse error", err)
	}
}

func TestAppConfigValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		AppEnv:          "development",
		LogLevel:        "info",
		HTTPAddr:        ":8080",
		MetricsAddr:     ":9092",
		DatabaseURL:     "postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		DefaultLimit:    150,
		MaxLimit:        100,
		MinimumMatches:  10,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
}
