package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rspassos/ilha/jobs/collector/internal/collector"
	"github.com/rspassos/ilha/jobs/collector/internal/config"
	"github.com/rspassos/ilha/jobs/collector/internal/httpclient"
	"github.com/rspassos/ilha/jobs/collector/internal/logging"
	"github.com/rspassos/ilha/jobs/collector/internal/metrics"
	"github.com/rspassos/ilha/jobs/collector/internal/storage"
)

type App struct {
	config        config.AppConfig
	bootstrapOnly bool
	logger        *logging.Logger
	metrics       *metrics.Collector
	service       *collector.Service
	pool          *storage.Pool
	metricsServer *http.Server
}

type Options struct {
	BootstrapOnly bool
	ConfigPath    string
	EnvFilePath   string
}

func NewApp(ctx context.Context, options Options) (*App, error) {
	cfg, err := config.NewLoader().Load(options.ConfigPath, options.EnvFilePath)
	if err != nil {
		return nil, err
	}

	app := &App{
		config:        cfg,
		bootstrapOnly: options.BootstrapOnly,
		logger:        logging.New(os.Stdout, "match-stats-collector"),
	}

	if options.BootstrapOnly {
		return app, nil
	}

	pool, err := storage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	repository := storage.NewRepository(pool)
	if err := repository.ApplyMigrations(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	metricsCollector := metrics.New()
	app.metrics = metricsCollector
	app.pool = pool
	app.metricsServer = &http.Server{
		Addr:    cfg.MetricsAddr,
		Handler: metricsCollector.Handler(),
	}

	client := httpclient.New(cfg.HubAPIBaseURL, nil)
	app.service = collector.NewService(client, client, repository, nil, app.logger, metricsCollector)

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.logger.Info("collector bootstrap completed", map[string]any{
		"app_env":        a.config.AppEnv,
		"log_level":      a.config.LogLevel,
		"metrics_addr":   a.config.MetricsAddr,
		"bootstrap_only": a.bootstrapOnly,
		"server_count":   len(a.config.Servers),
		"enabled_count":  len(a.config.EnabledServers()),
	}); err != nil {
		return err
	}

	if a.bootstrapOnly {
		return nil
	}

	if a.metricsServer != nil && a.config.MetricsAddr != "" {
		go func() {
			if err := a.metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				_ = a.logger.Error("metrics server failed", map[string]any{
					"error": err.Error(),
				})
			}
		}()
	}

	defer func() {
		if a.metricsServer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = a.metricsServer.Shutdown(shutdownCtx)
		}
		if a.pool != nil {
			a.pool.Close()
		}
	}()

	if err := a.logger.Info("collector run started", map[string]any{
		"enabled_servers": len(a.config.EnabledServers()),
	}); err != nil {
		return err
	}

	if err := a.service.RunOnce(ctx, a.config.EnabledServers()); err != nil {
		_ = a.logger.Error("collector run failed", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("run collector once: %w", err)
	}

	return nil
}
