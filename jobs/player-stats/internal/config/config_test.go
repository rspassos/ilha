package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderLoadReadsEnvFile(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("PLAYER_STATS_METRICS_ADDR", "")
	t.Setenv("PLAYER_STATS_BATCH_SIZE", "")
	t.Setenv("PLAYER_STATS_JOB_NAME", "")

	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envPath, []byte(`
DATABASE_URL=postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable
APP_ENV=local
LOG_LEVEL=debug
PLAYER_STATS_METRICS_ADDR=:9191
PLAYER_STATS_BATCH_SIZE=250
PLAYER_STATS_JOB_NAME=player-stats-local
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
	if cfg.MetricsAddr != ":9191" {
		t.Fatalf("MetricsAddr = %q, want :9191", cfg.MetricsAddr)
	}
	if cfg.BatchSize != 250 {
		t.Fatalf("BatchSize = %d, want 250", cfg.BatchSize)
	}
	if cfg.JobName != "player-stats-local" {
		t.Fatalf("JobName = %q, want player-stats-local", cfg.JobName)
	}
}

func TestLoaderLoadKeepsExistingEnvironmentValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://from-os")
	t.Setenv("PLAYER_STATS_BATCH_SIZE", "75")
	t.Setenv("PLAYER_STATS_JOB_NAME", "from-os")

	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envPath, []byte(`
DATABASE_URL=postgres://from-file
PLAYER_STATS_BATCH_SIZE=25
PLAYER_STATS_JOB_NAME=from-file
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
	if cfg.BatchSize != 75 {
		t.Fatalf("BatchSize = %d, want 75", cfg.BatchSize)
	}
	if cfg.JobName != "from-os" {
		t.Fatalf("JobName = %q, want from-os", cfg.JobName)
	}
}

func TestLoaderLoadRejectsInvalidBatchSize(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable")
	t.Setenv("PLAYER_STATS_BATCH_SIZE", "not-a-number")

	_, err := NewLoader().Load("")
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse player_stats_batch_size") {
		t.Fatalf("Load() error = %v, want batch size parse error", err)
	}
}

func TestAppConfigValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		AppEnv:      "development",
		LogLevel:    "info",
		MetricsAddr: ":9091",
		DatabaseURL: "postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable",
		BatchSize:   0,
		JobName:     "player-stats",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
}
