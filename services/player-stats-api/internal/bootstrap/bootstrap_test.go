package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rspassos/ilha/services/player-stats-api/internal/config"
	"github.com/rspassos/ilha/services/player-stats-api/internal/httpapi"
	"github.com/rspassos/ilha/services/player-stats-api/internal/logging"
	"github.com/rspassos/ilha/services/player-stats-api/internal/metrics"
	"github.com/rspassos/ilha/services/player-stats-api/internal/model"
	"github.com/rspassos/ilha/services/player-stats-api/internal/service"
)

func TestNewAppLoadsEnvConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("PLAYER_STATS_API_HTTP_ADDR", "")
	t.Setenv("PLAYER_STATS_API_METRICS_ADDR", "")
	t.Setenv("PLAYER_STATS_API_DEFAULT_LIMIT", "")
	t.Setenv("PLAYER_STATS_API_MAX_LIMIT", "")
	t.Setenv("PLAYER_STATS_API_MINIMUM_MATCHES", "")

	tempDir := t.TempDir()
	envFilePath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envFilePath, []byte(`
DATABASE_URL=postgres://ilha:ilha@postgres:5432/ilha?sslmode=disable
APP_ENV=test
LOG_LEVEL=debug
PLAYER_STATS_API_HTTP_ADDR=:18080
PLAYER_STATS_API_METRICS_ADDR=:19092
PLAYER_STATS_API_DEFAULT_LIMIT=25
PLAYER_STATS_API_MAX_LIMIT=250
PLAYER_STATS_API_MINIMUM_MATCHES=11
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
	if app.config.HTTPAddr != ":18080" {
		t.Fatalf("HTTPAddr = %q, want :18080", app.config.HTTPAddr)
	}
	if app.config.DefaultLimit != 25 {
		t.Fatalf("DefaultLimit = %d, want 25", app.config.DefaultLimit)
	}
	if app.config.MaxLimit != 250 {
		t.Fatalf("MaxLimit = %d, want 250", app.config.MaxLimit)
	}
	if app.config.MinimumMatches != 11 {
		t.Fatalf("MinimumMatches = %d, want 11", app.config.MinimumMatches)
	}
}

func TestNewAppRejectsInvalidDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("PLAYER_STATS_API_HTTP_ADDR", "")
	t.Setenv("PLAYER_STATS_API_METRICS_ADDR", "")

	tempDir := t.TempDir()
	envFilePath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envFilePath, []byte(`
DATABASE_URL=://bad
PLAYER_STATS_API_HTTP_ADDR=127.0.0.1:0
PLAYER_STATS_API_METRICS_ADDR=127.0.0.1:0
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if _, err := NewApp(context.Background(), Options{
		EnvFilePath: envFilePath,
	}); err == nil {
		t.Fatal("NewApp() error = nil, want non-nil")
	}
}

func TestAppRunStartsAndStopsWithDatabase(t *testing.T) {
	databaseURL := os.Getenv("PLAYER_STATS_API_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("PLAYER_STATS_API_TEST_DATABASE_URL is not set")
	}

	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("PLAYER_STATS_API_HTTP_ADDR", "")
	t.Setenv("PLAYER_STATS_API_METRICS_ADDR", "")
	t.Setenv("PLAYER_STATS_API_SHUTDOWN_TIMEOUT", "")

	tempDir := t.TempDir()
	envFilePath := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(envFilePath, []byte(`
DATABASE_URL=`+databaseURL+`
APP_ENV=test
PLAYER_STATS_API_HTTP_ADDR=127.0.0.1:0
PLAYER_STATS_API_METRICS_ADDR=127.0.0.1:0
PLAYER_STATS_API_SHUTDOWN_TIMEOUT=2s
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	app, err := NewApp(context.Background(), Options{
		EnvFilePath: envFilePath,
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run(runCtx)
	}()

	httpClient := &http.Client{Timeout: 2 * time.Second}
	healthURL := "http://" + app.apiListener.Addr().String() + "/healthz"
	deadline := time.Now().Add(5 * time.Second)

	for {
		if time.Now().After(deadline) {
			t.Fatalf("health endpoint did not become ready: %s", healthURL)
		}

		resp, err := httpClient.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after cancellation")
	}
}

func TestNewHTTPServerServesRankingEndpoint(t *testing.T) {
	t.Parallel()

	var gotQuery model.RankingQuery
	server := newHTTPServer(config.AppConfig{
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		IdleTimeout:    time.Second,
		DefaultLimit:   25,
		MaxLimit:       100,
		MinimumMatches: 10,
	}, logging.New(io.Discard, "player-stats-api"), metrics.New(), service.RankingServiceFunc(func(_ context.Context, query model.RankingQuery) (model.RankingPage, error) {
		gotQuery = query
		return model.NewRankingPage(query, []model.PlayerRankingRow{
			{
				PlayerID:    "player-1",
				DisplayName: "Player One",
				Matches:     14,
				Efficiency:  58.2,
				Frags:       220,
				Kills:       200,
				Deaths:      144,
				LGAccuracy:  41.3,
				RLHits:      92,
				Rank:        1,
			},
		}, true), nil
	}))

	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/v1/rankings/players?mode=2on2&sort_by=frags&sort_direction=asc&limit=5&offset=10")
	if err != nil {
		t.Fatalf("GET ranking endpoint: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.StatusCode; got != http.StatusOK {
		t.Fatalf("status = %d, want %d", got, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	if gotQuery.Mode != "2on2" {
		t.Fatalf("service query mode = %q, want 2on2", gotQuery.Mode)
	}
	if gotQuery.SortBy != "frags" {
		t.Fatalf("service query sort_by = %q, want frags", gotQuery.SortBy)
	}
	if gotQuery.SortDirection != "asc" {
		t.Fatalf("service query sort_direction = %q, want asc", gotQuery.SortDirection)
	}
	if gotQuery.Limit != 5 {
		t.Fatalf("service query limit = %d, want 5", gotQuery.Limit)
	}
	if gotQuery.Offset != 10 {
		t.Fatalf("service query offset = %d, want 10", gotQuery.Offset)
	}

	var page model.RankingPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(page.Data))
	}
	if page.Meta.SortBy != "frags" {
		t.Fatalf("meta.sort_by = %q, want frags", page.Meta.SortBy)
	}
	if page.Meta.SortDirection != "asc" {
		t.Fatalf("meta.sort_direction = %q, want asc", page.Meta.SortDirection)
	}
	if page.Meta.Limit != 5 {
		t.Fatalf("meta.limit = %d, want 5", page.Meta.Limit)
	}
	if page.Meta.Offset != 10 {
		t.Fatalf("meta.offset = %d, want 10", page.Meta.Offset)
	}
	if !page.Meta.HasNext {
		t.Fatal("meta.has_next = false, want true")
	}
}

func TestNewHTTPServerServesEmptyRankingPage(t *testing.T) {
	t.Parallel()

	server := newHTTPServer(config.AppConfig{
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		IdleTimeout:    time.Second,
		DefaultLimit:   25,
		MaxLimit:       100,
		MinimumMatches: 10,
	}, logging.New(io.Discard, "player-stats-api"), metrics.New(), service.RankingServiceFunc(func(_ context.Context, query model.RankingQuery) (model.RankingPage, error) {
		return model.NewRankingPage(query, []model.PlayerRankingRow{}, false), nil
	}))

	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/v1/rankings/players")
	if err != nil {
		t.Fatalf("GET ranking endpoint: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.StatusCode; got != http.StatusOK {
		t.Fatalf("status = %d, want %d", got, http.StatusOK)
	}

	var page model.RankingPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if len(page.Data) != 0 {
		t.Fatalf("data len = %d, want 0", len(page.Data))
	}
	if page.Meta.Returned != 0 {
		t.Fatalf("meta.returned = %d, want 0", page.Meta.Returned)
	}
	if page.Meta.Limit != 25 {
		t.Fatalf("meta.limit = %d, want 25", page.Meta.Limit)
	}
	if page.Meta.SortBy != model.DefaultSortBy {
		t.Fatalf("meta.sort_by = %q, want %q", page.Meta.SortBy, model.DefaultSortBy)
	}
	if page.Meta.SortDirection != model.DefaultSortDirection {
		t.Fatalf("meta.sort_direction = %q, want %q", page.Meta.SortDirection, model.DefaultSortDirection)
	}
}

func TestNewHTTPServerReturnsProblemJSONForInvalidQuery(t *testing.T) {
	t.Parallel()

	server := newHTTPServer(config.AppConfig{
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		IdleTimeout:    time.Second,
		DefaultLimit:   25,
		MaxLimit:       100,
		MinimumMatches: 10,
	}, logging.New(io.Discard, "player-stats-api"), metrics.New(), httpapi.NewNoopRankingService())

	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/v1/rankings/players?sort_by=kills")
	if err != nil {
		t.Fatalf("GET ranking endpoint: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.StatusCode; got != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", got, http.StatusBadRequest)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", got)
	}

	var problem httpapi.Problem
	if err := json.NewDecoder(resp.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem body: %v", err)
	}
	if problem.Title != "Invalid query parameters" {
		t.Fatalf("title = %q, want Invalid query parameters", problem.Title)
	}
	if len(problem.InvalidParams) != 1 {
		t.Fatalf("invalid_params len = %d, want 1", len(problem.InvalidParams))
	}
}

func TestNewHTTPServerReturnsProblemJSONForInternalError(t *testing.T) {
	t.Parallel()

	server := newHTTPServer(config.AppConfig{
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		IdleTimeout:    time.Second,
		DefaultLimit:   25,
		MaxLimit:       100,
		MinimumMatches: 10,
	}, logging.New(io.Discard, "player-stats-api"), metrics.New(), service.RankingServiceFunc(func(context.Context, model.RankingQuery) (model.RankingPage, error) {
		return model.RankingPage{}, errors.New("boom")
	}))

	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/v1/rankings/players")
	if err != nil {
		t.Fatalf("GET ranking endpoint: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.StatusCode; got != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", got, http.StatusInternalServerError)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", got)
	}
}

func TestInstrumentHandlerLogsCompletedRequestWithRankingFields(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	server := newHTTPServer(config.AppConfig{
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		IdleTimeout:    time.Second,
		DefaultLimit:   25,
		MaxLimit:       100,
		MinimumMatches: 10,
	}, logging.New(&logBuffer, "player-stats-api"), metrics.New(), service.RankingServiceFunc(func(_ context.Context, query model.RankingQuery) (model.RankingPage, error) {
		return model.NewRankingPage(query, []model.PlayerRankingRow{
			{PlayerID: "player-1", DisplayName: "Player One", Matches: 12, Rank: 1},
		}, false), nil
	}))

	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/v1/rankings/players?mode=2on2&limit=1")
	if err != nil {
		t.Fatalf("GET ranking endpoint: %v", err)
	}
	resp.Body.Close()

	output := logBuffer.String()
	for _, fragment := range []string{
		`"message":"player stats api request completed"`,
		`"endpoint":"/v1/rankings/players"`,
		`"status":200`,
		`"returned_rows":1`,
		`"sort_by":"efficiency"`,
		`"mode":"2on2"`,
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("log output missing %q in %s", fragment, output)
		}
	}
}
