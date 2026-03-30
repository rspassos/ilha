package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rspassos/ilha/jobs/player-stats/internal/config"
	"github.com/rspassos/ilha/jobs/player-stats/internal/logging"
	"github.com/rspassos/ilha/jobs/player-stats/internal/metrics"
	"github.com/rspassos/ilha/jobs/player-stats/internal/service"
	"github.com/rspassos/ilha/jobs/player-stats/internal/source"
	"github.com/rspassos/ilha/jobs/player-stats/internal/storage"
)

type App struct {
	config        config.AppConfig
	bootstrapOnly bool
	logger        *logging.Logger
	metrics       *metrics.Collector
	service       *service.Service
	repository    *storage.Repository
	pool          *storage.Pool
	metricsServer *http.Server
}

type Options struct {
	BootstrapOnly bool
	EnvFilePath   string
}

func NewApp(ctx context.Context, options Options) (*App, error) {
	cfg, err := config.NewLoader().Load(options.EnvFilePath)
	if err != nil {
		return nil, err
	}

	app := &App{
		config:        cfg,
		bootstrapOnly: options.BootstrapOnly,
		logger:        logging.New(os.Stdout, "player-stats"),
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
	app.repository = repository
	app.service = service.New(
		app.logger,
		metricsCollector,
		repository,
		source.NewPostgresSource(pool),
		cfg.JobName,
		cfg.BatchSize,
	)
	app.metricsServer = &http.Server{
		Addr:    cfg.MetricsAddr,
		Handler: metricsCollector.Handler(),
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.logger.Info("player stats bootstrap completed", map[string]any{
		"app_env":        a.config.AppEnv,
		"log_level":      a.config.LogLevel,
		"metrics_addr":   a.config.MetricsAddr,
		"bootstrap_only": a.bootstrapOnly,
		"batch_size":     a.config.BatchSize,
		"job_name":       a.config.JobName,
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

	if err := a.logger.Info("player stats run started", map[string]any{
		"job_name":   a.config.JobName,
		"batch_size": a.config.BatchSize,
	}); err != nil {
		return err
	}

	startedAt := time.Now()
	if a.metrics != nil {
		a.metrics.RecordRun("started")
	}

	if err := a.service.RunOnce(ctx); err != nil {
		if a.metrics != nil {
			a.metrics.RecordRun("error")
			a.metrics.ObserveStage("run_once", startedAt)
		}
		_ = a.logger.Error("player stats run failed", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("run player stats once: %w", err)
	}

	if a.metrics != nil {
		a.metrics.RecordRun("success")
		a.metrics.ObserveStage("run_once", startedAt)
	}

	return nil
}
