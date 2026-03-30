package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultEnvFile     = ".env"
	DefaultMetricsAddr = ":9091"
	DefaultBatchSize   = 100
	DefaultJobName     = "player-stats"
)

type Loader struct {
	lookupEnv func(string) (string, bool)
	readFile  func(string) ([]byte, error)
}

type AppConfig struct {
	AppEnv      string
	LogLevel    string
	MetricsAddr string
	DatabaseURL string
	BatchSize   int
	JobName     string
}

func NewLoader() Loader {
	return Loader{
		lookupEnv: os.LookupEnv,
		readFile:  os.ReadFile,
	}
}

func (l Loader) Load(envFilePath string) (AppConfig, error) {
	if l.lookupEnv == nil {
		l.lookupEnv = os.LookupEnv
	}
	if l.readFile == nil {
		l.readFile = os.ReadFile
	}
	if envFilePath == "" {
		envFilePath = DefaultEnvFile
	}

	if err := l.loadDotEnv(envFilePath); err != nil {
		return AppConfig{}, err
	}

	batchSize, err := envIntOrDefault(l.lookupEnv, "PLAYER_STATS_BATCH_SIZE", DefaultBatchSize)
	if err != nil {
		return AppConfig{}, err
	}

	cfg := AppConfig{
		AppEnv:      envOrDefault(l.lookupEnv, "APP_ENV", "development"),
		LogLevel:    envOrDefault(l.lookupEnv, "LOG_LEVEL", "info"),
		MetricsAddr: envOrDefault(l.lookupEnv, "PLAYER_STATS_METRICS_ADDR", DefaultMetricsAddr),
		DatabaseURL: strings.TrimSpace(envValue(l.lookupEnv, "DATABASE_URL")),
		BatchSize:   batchSize,
		JobName:     envOrDefault(l.lookupEnv, "PLAYER_STATS_JOB_NAME", DefaultJobName),
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

func (c AppConfig) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("database_url must not be empty")
	}
	parsedDatabaseURL, err := url.Parse(strings.TrimSpace(c.DatabaseURL))
	if err != nil {
		return fmt.Errorf("parse database_url: %w", err)
	}
	if parsedDatabaseURL.Scheme == "" || parsedDatabaseURL.Host == "" {
		return fmt.Errorf("database_url %q must include scheme and host", c.DatabaseURL)
	}
	if strings.TrimSpace(c.AppEnv) == "" {
		return errors.New("app_env must not be empty")
	}
	if strings.TrimSpace(c.LogLevel) == "" {
		return errors.New("log_level must not be empty")
	}
	if c.BatchSize <= 0 {
		return errors.New("player_stats_batch_size must be greater than zero")
	}
	if strings.TrimSpace(c.JobName) == "" {
		return errors.New("player_stats_job_name must not be empty")
	}

	return nil
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

func envIntOrDefault(lookupEnv func(string) (string, bool), key string, fallback int) (int, error) {
	value := strings.TrimSpace(envValue(lookupEnv, key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", strings.ToLower(key), err)
	}

	return parsed, nil
}
