package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAppLoadsEnvConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("PLAYER_STATS_BATCH_SIZE", "")
	t.Setenv("PLAYER_STATS_JOB_NAME", "")

	tempDir := t.TempDir()
	envFilePath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envFilePath, []byte(`
DATABASE_URL=postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable
APP_ENV=test
LOG_LEVEL=debug
PLAYER_STATS_BATCH_SIZE=50
PLAYER_STATS_JOB_NAME=player-stats-test
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	app, err := NewApp(context.Background(), Options{
		BootstrapOnly: true,
		EnvFilePath:   envFilePath,
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if app.config.AppEnv != "test" {
		t.Fatalf("AppEnv = %q, want test", app.config.AppEnv)
	}
	if app.config.BatchSize != 50 {
		t.Fatalf("BatchSize = %d, want 50", app.config.BatchSize)
	}
	if app.config.JobName != "player-stats-test" {
		t.Fatalf("JobName = %q, want player-stats-test", app.config.JobName)
	}
}

func TestNewAppRejectsMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("PLAYER_STATS_BATCH_SIZE", "")
	t.Setenv("PLAYER_STATS_JOB_NAME", "")

	tempDir := t.TempDir()
	envFilePath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envFilePath, []byte("APP_ENV=test\n"), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if _, err := NewApp(context.Background(), Options{
		BootstrapOnly: true,
		EnvFilePath:   envFilePath,
	}); err == nil {
		t.Fatal("NewApp() error = nil, want non-nil")
	}
}
