package storage

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rspassos/ilha/jobs/collector/internal/model"
)

func TestNullHelpers(t *testing.T) {
	t.Parallel()

	if value := nullIfEmpty("   "); value != nil {
		t.Fatalf("nullIfEmpty whitespace = %#v, want nil", value)
	}
	if value := nullIfEmpty("abc"); value != "abc" {
		t.Fatalf("nullIfEmpty abc = %#v, want abc", value)
	}
	if value := nullInt(0); value != nil {
		t.Fatalf("nullInt(0) = %#v, want nil", value)
	}
	if value := nullInt(42); value != 42 {
		t.Fatalf("nullInt(42) = %#v, want 42", value)
	}
}

func TestMigrationSQLContainsExpectedIndexes(t *testing.T) {
	t.Parallel()

	data, err := migrationFiles.ReadFile("migrations/001_create_collector_matches.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	sql := string(data)
	for _, snippet := range []string{
		"CREATE TABLE IF NOT EXISTS collector_matches",
		"CONSTRAINT collector_matches_server_demo_key UNIQUE (server_key, demo_name)",
		"collector_matches_mode_played_at_idx",
		"collector_matches_has_bots_played_at_idx",
		"score_payload jsonb NOT NULL",
		"merged_payload jsonb NOT NULL",
	} {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("migration SQL missing %q", snippet)
		}
	}
}

func TestRepositoryUpsertMatchesIntegration(t *testing.T) {
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

	repository := NewRepository(pool)
	repository.now = func() time.Time {
		return time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	}

	if _, err := pool.Exec(ctx, "DROP TABLE IF EXISTS collector_matches"); err != nil {
		t.Fatalf("drop collector_matches: %v", err)
	}
	if _, err := pool.Exec(ctx, "DROP TABLE IF EXISTS collector_schema_migrations"); err != nil {
		t.Fatalf("drop collector_schema_migrations: %v", err)
	}
	if err := repository.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	first := testMatchRecord(t)
	result, err := repository.UpsertMatches(ctx, []model.MatchRecord{first})
	if err != nil {
		t.Fatalf("UpsertMatches() insert error = %v", err)
	}
	if result.Inserted != 1 || result.Updated != 0 {
		t.Fatalf("first UpsertMatches() result = %#v", result)
	}

	first.MapName = "aerowalk"
	first.HasBots = true
	secondResult, err := repository.UpsertMatches(ctx, []model.MatchRecord{first})
	if err != nil {
		t.Fatalf("UpsertMatches() update error = %v", err)
	}
	if secondResult.Inserted != 0 || secondResult.Updated != 1 {
		t.Fatalf("second UpsertMatches() result = %#v", secondResult)
	}

	var mapName string
	var hasBots bool
	var mergedPayload []byte
	if err := pool.QueryRow(ctx, `SELECT map_name, has_bots, merged_payload FROM collector_matches WHERE server_key = $1 AND demo_name = $2`, first.ServerKey, first.DemoName).Scan(&mapName, &hasBots, &mergedPayload); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if mapName != "aerowalk" {
		t.Fatalf("map_name = %q, want aerowalk", mapName)
	}
	if !hasBots {
		t.Fatal("has_bots = false, want true")
	}
	if len(mergedPayload) == 0 {
		t.Fatal("merged_payload is empty")
	}
}

func testMatchRecord(t *testing.T) model.MatchRecord {
	t.Helper()

	payload, err := json.Marshal(map[string]any{"demo": "demo-1.mvd"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	return model.MatchRecord{
		ServerKey:       "qlash-br-1",
		ServerName:      "Qlash Brazil 1",
		DemoName:        "demo-1.mvd",
		MatchKey:        "qlash-br-1:demo-1.mvd",
		Mode:            "duel",
		MapName:         "dm6",
		Participants:    "alpha vs beta",
		PlayedAt:        time.Date(2026, 3, 19, 23, 0, 0, 0, time.UTC),
		DurationSeconds: 600,
		Hostname:        "test-host",
		HasBots:         false,
		ScorePayload:    payload,
		StatsPayload:    payload,
		MergedPayload:   payload,
	}
}
