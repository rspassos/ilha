package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAppLoadsConfigAndEnv(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "collector.yaml")
	envFilePath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(configPath, []byte(`
servers:
  - key: qlash-br-1
    name: Qlash Brazil 1
    address: qw.qlash.com.br:28501
    enabled: true
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := os.WriteFile(envFilePath, []byte(`
DATABASE_URL=postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable
HUBAPI_BASE_URL=https://hubapi.quakeworld.nu
APP_ENV=test
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	app, err := NewApp(nil, Options{
		BootstrapOnly: true,
		ConfigPath:    configPath,
		EnvFilePath:   envFilePath,
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if app.config.AppEnv != "test" {
		t.Fatalf("AppEnv = %q, want test", app.config.AppEnv)
	}
	if len(app.config.EnabledServers()) != 1 {
		t.Fatalf("EnabledServers len = %d, want 1", len(app.config.EnabledServers()))
	}
}
