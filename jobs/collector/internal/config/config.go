package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v4"
)

const (
	DefaultConfigPath = "config/collector.yaml"
	DefaultEnvFile    = ".env"
)

type Loader struct {
	lookupEnv func(string) (string, bool)
	readFile  func(string) ([]byte, error)
}

type AppConfig struct {
	AppEnv        string
	LogLevel      string
	MetricsAddr   string
	DatabaseURL   string
	HubAPIBaseURL string
	Servers       []ServerConfig
}

type FileConfig struct {
	Servers []ServerConfig `yaml:"servers"`
}

type ServerConfig struct {
	Key            string `yaml:"key"`
	Name           string `yaml:"name"`
	Address        string `yaml:"address"`
	Enabled        bool   `yaml:"enabled"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

func NewLoader() Loader {
	return Loader{
		lookupEnv: os.LookupEnv,
		readFile:  os.ReadFile,
	}
}

func (l Loader) Load(configPath string, envFilePath string) (AppConfig, error) {
	if l.lookupEnv == nil {
		l.lookupEnv = os.LookupEnv
	}
	if l.readFile == nil {
		l.readFile = os.ReadFile
	}

	if envFilePath == "" {
		envFilePath = DefaultEnvFile
	}
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	if err := l.loadDotEnv(envFilePath); err != nil {
		return AppConfig{}, err
	}

	fileConfig, err := l.loadFileConfig(configPath)
	if err != nil {
		return AppConfig{}, err
	}

	cfg := AppConfig{
		AppEnv:        envOrDefault(l.lookupEnv, "APP_ENV", "development"),
		LogLevel:      envOrDefault(l.lookupEnv, "LOG_LEVEL", "info"),
		MetricsAddr:   envOrDefault(l.lookupEnv, "METRICS_ADDR", ":9090"),
		DatabaseURL:   strings.TrimSpace(envValue(l.lookupEnv, "DATABASE_URL")),
		HubAPIBaseURL: strings.TrimSpace(envValue(l.lookupEnv, "HUBAPI_BASE_URL")),
		Servers:       fileConfig.Servers,
	}

	if err := cfg.Validate(); err != nil {
		return AppConfig{}, err
	}

	return cfg, nil
}

func (l Loader) loadDotEnv(path string) error {
	data, err := l.readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read env file %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	for index, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("parse env file %s line %d: missing '='", path, index+1)
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			return fmt.Errorf("parse env file %s line %d: empty key", path, index+1)
		}
		if currentValue, exists := l.lookupEnv(key); exists && strings.TrimSpace(currentValue) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env from %s line %d: %w", path, index+1, err)
		}
	}

	return nil
}

func (l Loader) loadFileConfig(path string) (FileConfig, error) {
	data, err := l.readFile(path)
	if err != nil {
		return FileConfig{}, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg FileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return FileConfig{}, fmt.Errorf("parse config file %s: %w", path, err)
	}

	for index := range cfg.Servers {
		cfg.Servers[index].applyDefaults()
		if err := cfg.Servers[index].Validate(); err != nil {
			return FileConfig{}, fmt.Errorf("validate server %d in %s: %w", index, path, err)
		}
	}

	return cfg, nil
}

func (c AppConfig) EnabledServers() []ServerConfig {
	servers := make([]ServerConfig, 0, len(c.Servers))
	for _, server := range c.Servers {
		if server.Enabled {
			servers = append(servers, server)
		}
	}
	return servers
}

func (c AppConfig) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("database_url must not be empty")
	}
	if strings.TrimSpace(c.HubAPIBaseURL) == "" {
		return errors.New("hubapi_base_url must not be empty")
	}
	parsedHubAPIBaseURL, err := url.Parse(strings.TrimSpace(c.HubAPIBaseURL))
	if err != nil {
		return fmt.Errorf("parse hubapi_base_url: %w", err)
	}
	if parsedHubAPIBaseURL.Scheme == "" || parsedHubAPIBaseURL.Host == "" {
		return fmt.Errorf("hubapi_base_url %q must include scheme and host", c.HubAPIBaseURL)
	}
	if strings.TrimSpace(c.AppEnv) == "" {
		return errors.New("app_env must not be empty")
	}
	if strings.TrimSpace(c.LogLevel) == "" {
		return errors.New("log_level must not be empty")
	}
	if len(c.Servers) == 0 {
		return errors.New("at least one server must be configured")
	}

	seen := make(map[string]struct{}, len(c.Servers))
	for _, server := range c.Servers {
		if _, exists := seen[server.Key]; exists {
			return fmt.Errorf("duplicate server key %q", server.Key)
		}
		seen[server.Key] = struct{}{}
	}

	return nil
}

func (s ServerConfig) Validate() error {
	if strings.TrimSpace(s.Key) == "" {
		return errors.New("key must not be empty")
	}
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("name must not be empty")
	}
	if strings.TrimSpace(s.Address) == "" {
		return errors.New("address must not be empty")
	}
	if s.TimeoutSeconds <= 0 {
		return errors.New("timeout_seconds must be greater than zero")
	}
	return nil
}

func (s *ServerConfig) applyDefaults() {
	if s.TimeoutSeconds == 0 {
		s.TimeoutSeconds = 5
	}
}

func envOrDefault(lookupEnv func(string) (string, bool), key string, fallback string) string {
	value := strings.TrimSpace(envValue(lookupEnv, key))
	if value == "" {
		return fallback
	}
	return value
}

func envValue(lookupEnv func(string) (string, bool), key string) string {
	value, _ := lookupEnv(key)
	return value
}

func ResolvePath(baseDir string, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}
