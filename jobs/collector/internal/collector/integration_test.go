package collector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rspassos/ilha/jobs/collector/internal/config"
	"github.com/rspassos/ilha/jobs/collector/internal/httpclient"
	"github.com/rspassos/ilha/jobs/collector/internal/logging"
	"github.com/rspassos/ilha/jobs/collector/internal/merge"
	"github.com/rspassos/ilha/jobs/collector/internal/metrics"
	"github.com/rspassos/ilha/jobs/collector/internal/storage"
)

func TestServiceRunOnceIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("COLLECTOR_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("COLLECTOR_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer pool.Close()

	lockConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer lockConn.Release()
	if _, err := lockConn.Exec(ctx, "SELECT pg_advisory_lock($1)", int64(505001)); err != nil {
		t.Fatalf("pg_advisory_lock() error = %v", err)
	}
	defer func() {
		_, _ = lockConn.Exec(ctx, "SELECT pg_advisory_unlock($1)", int64(505001))
	}()

	repository := storage.NewRepository(pool)
	if _, err := pool.Exec(ctx, "DROP TABLE IF EXISTS collector_matches"); err != nil {
		t.Fatalf("drop collector_matches: %v", err)
	}
	if _, err := pool.Exec(ctx, "DROP TABLE IF EXISTS collector_schema_migrations"); err != nil {
		t.Fatalf("drop collector_schema_migrations: %v", err)
	}
	if err := repository.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	server := newFixtureServer(t)
	service := NewService(
		httpclient.New(server.URL, nil),
		httpclient.New(server.URL, nil),
		repository,
		merge.New(),
		logging.New(ioDiscard{}, "match-stats-collector"),
		metrics.New(),
	)

	cfg := config.ServerConfig{
		Key:            "qlash-br-1",
		Name:           "Qlash Brazil 1",
		Address:        "qlash-br-1:27500",
		Enabled:        true,
		TimeoutSeconds: 5,
	}
	if err := service.RunOnce(ctx, []config.ServerConfig{cfg}); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM collector_matches").Scan(&count); err != nil {
		t.Fatalf("count collector_matches: %v", err)
	}
	if count != 9 {
		t.Fatalf("collector_matches count = %d, want 9", count)
	}
}

func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()

	lastScores := mustReadRepoFile(t, "../../../../docs/responses/lastscores.json")
	lastStats := mustReadRepoFile(t, "../../../../docs/responses/laststats.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/servers/qlash-br-1:27500/lastscores":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(lastScores)
		case "/v2/servers/qlash-br-1:27500/laststats":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(lastStats)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func mustReadRepoFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(relativePath))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}
	return data
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
