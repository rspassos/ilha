package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultEnvFile        = ".env"
	DefaultHTTPAddr       = ":8080"
	DefaultMetricsAddr    = ":9092"
	DefaultReadTimeout    = 5 * time.Second
	DefaultWriteTimeout   = 10 * time.Second
	DefaultIdleTimeout    = 60 * time.Second
	DefaultShutdownTimout = 5 * time.Second
	DefaultLimit          = 50
	DefaultMaxLimit       = 100
	DefaultMinimumMatches = 10
)

type Loader struct {
	lookupEnv func(string) (string, bool)
	readFile  func(string) ([]byte, error)
}

type AppConfig struct {
	AppEnv          string
	LogLevel        string
	HTTPAddr        string
	MetricsAddr     string
	DatabaseURL     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	DefaultLimit    int
	MaxLimit        int
	MinimumMatches  int
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

	readTimeout, err := envDurationOrDefault(l.lookupEnv, "PLAYER_STATS_API_READ_TIMEOUT", DefaultReadTimeout)
	if err != nil {
		return AppConfig{}, err
	}
	writeTimeout, err := envDurationOrDefault(l.lookupEnv, "PLAYER_STATS_API_WRITE_TIMEOUT", DefaultWriteTimeout)
	if err != nil {
		return AppConfig{}, err
	}
	idleTimeout, err := envDurationOrDefault(l.lookupEnv, "PLAYER_STATS_API_IDLE_TIMEOUT", DefaultIdleTimeout)
	if err != nil {
		return AppConfig{}, err
	}
	shutdownTimeout, err := envDurationOrDefault(l.lookupEnv, "PLAYER_STATS_API_SHUTDOWN_TIMEOUT", DefaultShutdownTimout)
	if err != nil {
		return AppConfig{}, err
	}
	defaultLimit, err := envIntOrDefault(l.lookupEnv, "PLAYER_STATS_API_DEFAULT_LIMIT", DefaultLimit)
	if err != nil {
		return AppConfig{}, err
	}
	maxLimit, err := envIntOrDefault(l.lookupEnv, "PLAYER_STATS_API_MAX_LIMIT", DefaultMaxLimit)
	if err != nil {
		return AppConfig{}, err
	}
	minimumMatches, err := envIntOrDefault(l.lookupEnv, "PLAYER_STATS_API_MINIMUM_MATCHES", DefaultMinimumMatches)
	if err != nil {
		return AppConfig{}, err
	}

	cfg := AppConfig{
		AppEnv:          envOrDefault(l.lookupEnv, "APP_ENV", "development"),
		LogLevel:        envOrDefault(l.lookupEnv, "LOG_LEVEL", "info"),
		HTTPAddr:        envOrDefault(l.lookupEnv, "PLAYER_STATS_API_HTTP_ADDR", DefaultHTTPAddr),
		MetricsAddr:     envOrDefault(l.lookupEnv, "PLAYER_STATS_API_METRICS_ADDR", DefaultMetricsAddr),
		DatabaseURL:     strings.TrimSpace(envValue(l.lookupEnv, "DATABASE_URL")),
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ShutdownTimeout: shutdownTimeout,
		DefaultLimit:    defaultLimit,
		MaxLimit:        maxLimit,
		MinimumMatches:  minimumMatches,
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
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return errors.New("player_stats_api_http_addr must not be empty")
	}
	if strings.TrimSpace(c.MetricsAddr) == "" {
		return errors.New("player_stats_api_metrics_addr must not be empty")
	}
	if c.ReadTimeout <= 0 {
		return errors.New("player_stats_api_read_timeout must be greater than zero")
	}
	if c.WriteTimeout <= 0 {
		return errors.New("player_stats_api_write_timeout must be greater than zero")
	}
	if c.IdleTimeout <= 0 {
		return errors.New("player_stats_api_idle_timeout must be greater than zero")
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("player_stats_api_shutdown_timeout must be greater than zero")
	}
	if c.DefaultLimit <= 0 {
		return errors.New("player_stats_api_default_limit must be greater than zero")
	}
	if c.MaxLimit <= 0 {
		return errors.New("player_stats_api_max_limit must be greater than zero")
	}
	if c.DefaultLimit > c.MaxLimit {
		return errors.New("player_stats_api_default_limit must be less than or equal to player_stats_api_max_limit")
	}
	if c.MinimumMatches <= 0 {
		return errors.New("player_stats_api_minimum_matches must be greater than zero")
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

func envDurationOrDefault(lookupEnv func(string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(envValue(lookupEnv, key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", strings.ToLower(key), err)
	}

	return parsed, nil
}
