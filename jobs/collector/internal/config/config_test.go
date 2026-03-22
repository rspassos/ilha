package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderLoadReadsYAMLAndEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "collector.yaml")
	envPath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(configPath, []byte(`
servers:
  - key: qlash-br-1
    name: Qlash Brazil 1
    address: qw.qlash.com.br:28501
    enabled: true
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := os.WriteFile(envPath, []byte(`
DATABASE_URL=postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable
HUBAPI_BASE_URL=https://hubapi.quakeworld.nu
APP_ENV=local
LOG_LEVEL=debug
METRICS_ADDR=:9100
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("DATABASE_URL", "")
	t.Setenv("HUBAPI_BASE_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("METRICS_ADDR", "")

	cfg, err := NewLoader().Load(configPath, envPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.HubAPIBaseURL != "https://hubapi.quakeworld.nu" {
		t.Fatalf("HubAPIBaseURL = %q, want https://hubapi.quakeworld.nu", cfg.HubAPIBaseURL)
	}
	if cfg.AppEnv != "local" {
		t.Fatalf("AppEnv = %q, want local", cfg.AppEnv)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.MetricsAddr != ":9100" {
		t.Fatalf("MetricsAddr = %q, want :9100", cfg.MetricsAddr)
	}
	if len(cfg.Servers) != 1 {
		t.Fatalf("Servers len = %d, want 1", len(cfg.Servers))
	}
	if cfg.Servers[0].TimeoutSeconds != 5 {
		t.Fatalf("TimeoutSeconds = %d, want 5", cfg.Servers[0].TimeoutSeconds)
	}
}

func TestLoaderLoadKeepsExistingEnvironmentValues(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "collector.yaml")
	envPath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(configPath, []byte(`
servers:
  - key: qlash-br-1
    name: Qlash Brazil 1
    address: qw.qlash.com.br:28501
    enabled: true
    timeout_seconds: 10
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := os.WriteFile(envPath, []byte("DATABASE_URL=postgres://from-file\nHUBAPI_BASE_URL=https://from-file.example\n"), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("DATABASE_URL", "postgres://from-os")
	t.Setenv("HUBAPI_BASE_URL", "https://from-os.example")

	cfg, err := NewLoader().Load(configPath, envPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://from-os" {
		t.Fatalf("DatabaseURL = %q, want postgres://from-os", cfg.DatabaseURL)
	}
	if cfg.HubAPIBaseURL != "https://from-os.example" {
		t.Fatalf("HubAPIBaseURL = %q, want https://from-os.example", cfg.HubAPIBaseURL)
	}
}

func TestLoaderLoadRejectsInvalidYAMLConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "collector.yaml")

	if err := os.WriteFile(configPath, []byte(`
servers:
  - key: ""
    name: invalid
    address: qw.qlash.com.br:28501
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("DATABASE_URL", "postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable")
	t.Setenv("HUBAPI_BASE_URL", "https://hubapi.quakeworld.nu")

	_, err := NewLoader().Load(configPath, "")
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "validate server") {
		t.Fatalf("Load() error = %v, want validation error", err)
	}
}

func TestAppConfigValidateRejectsDuplicateServerKeys(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		AppEnv:        "development",
		LogLevel:      "info",
		MetricsAddr:   ":9090",
		DatabaseURL:   "postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable",
		HubAPIBaseURL: "https://hubapi.quakeworld.nu",
		Servers: []ServerConfig{
			{Key: "one", Name: "one", Address: "one:27500", Enabled: true, TimeoutSeconds: 5},
			{Key: "one", Name: "two", Address: "two:27500", Enabled: true, TimeoutSeconds: 5},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
}

func TestEnabledServersReturnsOnlyEnabledOnes(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		Servers: []ServerConfig{
			{Key: "one", Enabled: true},
			{Key: "two", Enabled: false},
			{Key: "three", Enabled: true},
		},
	}

	enabled := cfg.EnabledServers()
	if len(enabled) != 2 {
		t.Fatalf("EnabledServers len = %d, want 2", len(enabled))
	}
	if enabled[0].Key != "one" || enabled[1].Key != "three" {
		t.Fatalf("EnabledServers unexpected result: %#v", enabled)
	}
}
