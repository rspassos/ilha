package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rspassos/ilha/services/player-stats-api/internal/config"
	"github.com/rspassos/ilha/services/player-stats-api/internal/httpapi"
	"github.com/rspassos/ilha/services/player-stats-api/internal/logging"
	"github.com/rspassos/ilha/services/player-stats-api/internal/metrics"
	"github.com/rspassos/ilha/services/player-stats-api/internal/model"
	"github.com/rspassos/ilha/services/player-stats-api/internal/service"
	"github.com/rspassos/ilha/services/player-stats-api/internal/storage"
)

type App struct {
	config        config.AppConfig
	bootstrapOnly bool
	logger        *logging.Logger
	metrics       *metrics.Collector
	pool          *pgxpool.Pool
	apiServer     *http.Server
	apiListener   net.Listener
	metricsServer *http.Server
	metricsListen net.Listener
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
		logger:        logging.New(os.Stdout, "player-stats-api"),
	}

	if options.BootstrapOnly {
		return app, nil
	}

	pool, err := openPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	metricsCollector := metrics.New()
	apiListener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("listen http addr %s: %w", cfg.HTTPAddr, err)
	}

	metricsListener, err := net.Listen("tcp", cfg.MetricsAddr)
	if err != nil {
		_ = apiListener.Close()
		pool.Close()
		return nil, fmt.Errorf("listen metrics addr %s: %w", cfg.MetricsAddr, err)
	}

	app.metrics = metricsCollector
	app.pool = pool
	app.apiListener = apiListener
	app.metricsListen = metricsListener
	repository := storage.NewRepository((*storage.Pool)(pool), app.logger, metricsCollector)
	rankingService, err := service.NewRankingService(repository, service.RankingConfig{
		DefaultLimit:   cfg.DefaultLimit,
		MaxLimit:       cfg.MaxLimit,
		MinimumMatches: cfg.MinimumMatches,
		DefaultSortBy:  model.DefaultSortBy,
		DefaultSortDir: model.DefaultSortDirection,
	})
	if err != nil {
		_ = apiListener.Close()
		_ = metricsListener.Close()
		pool.Close()
		return nil, fmt.Errorf("build ranking service: %w", err)
	}

	app.apiServer = newHTTPServer(cfg, app.logger, metricsCollector, rankingService)
	app.metricsServer = &http.Server{
		Handler:           metricsCollector.Handler(),
		ReadHeaderTimeout: cfg.ReadTimeout,
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.logger.Info("player stats api bootstrap completed", map[string]any{
		"app_env":          a.config.AppEnv,
		"log_level":        a.config.LogLevel,
		"http_addr":        a.config.HTTPAddr,
		"metrics_addr":     a.config.MetricsAddr,
		"bootstrap_only":   a.bootstrapOnly,
		"default_limit":    a.config.DefaultLimit,
		"max_limit":        a.config.MaxLimit,
		"minimum_matches":  a.config.MinimumMatches,
		"shutdown_timeout": a.config.ShutdownTimeout.String(),
	}); err != nil {
		return err
	}

	if a.bootstrapOnly {
		return nil
	}

	if err := a.logger.Info("player stats api run started", map[string]any{
		"http_addr":    a.apiListener.Addr().String(),
		"metrics_addr": a.metricsListen.Addr().String(),
	}); err != nil {
		return err
	}

	serverErrCh := make(chan error, 2)

	go serve("http", a.apiServer, a.apiListener, serverErrCh)
	go serve("metrics", a.metricsServer, a.metricsListen, serverErrCh)

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-serverErrCh:
		_ = a.logger.Error("player stats api server failed", map[string]any{
			"error": runErr.Error(),
		})
	}

	shutdownErr := a.shutdown()
	if runErr != nil {
		return runErr
	}

	return shutdownErr
}

func (a *App) shutdown() error {
	var shutdownErr error
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
	defer cancel()

	if a.apiServer != nil {
		if err := a.apiServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("shutdown http server: %w", err))
		}
	}
	if a.metricsServer != nil {
		if err := a.metricsServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("shutdown metrics server: %w", err))
		}
	}
	if a.pool != nil {
		a.pool.Close()
	}

	return shutdownErr
}

func openPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func newHTTPServer(cfg config.AppConfig, logger *logging.Logger, collector *metrics.Collector, rankingService service.RankingService) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/healthz", instrumentHandler(logger, collector, "/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok\n")
	})))
	mux.Handle("/v1/rankings/players", instrumentHandler(logger, collector, "/v1/rankings/players", httpapi.NewHandler(
		cfg.DefaultLimit,
		cfg.MaxLimit,
		cfg.MinimumMatches,
		rankingService,
		logger,
		collector,
	)))

	return &http.Server{
		Handler:           mux,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: cfg.ReadTimeout,
	}
}

func instrumentHandler(logger *logging.Logger, collector *metrics.Collector, endpoint string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		ctx, requestFields := logging.WithRequestFields(r.Context())
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r.WithContext(ctx))

		if collector != nil {
			collector.RecordRequest(endpoint, fmt.Sprintf("%d", recorder.statusCode))
			collector.ObserveRequest(endpoint, startedAt)
		}
		if logger != nil {
			fields := map[string]any{
				"endpoint":    endpoint,
				"method":      r.Method,
				"status":      recorder.statusCode,
				"duration_ms": time.Since(startedAt).Milliseconds(),
			}
			for key, value := range requestFields.Snapshot() {
				fields[key] = value
			}
			_ = logger.Info("player stats api request completed", fields)
		}
	})
}

func serve(name string, server *http.Server, listener net.Listener, errCh chan<- error) {
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- fmt.Errorf("%s server: %w", name, err)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
