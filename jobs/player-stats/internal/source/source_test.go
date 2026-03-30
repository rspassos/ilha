package source

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
)

const integrationDBLockID int64 = 505100

func TestClassifySkipReason(t *testing.T) {
	t.Parallel()

	validPayload := mustJSON(t, map[string]any{
		"demo": "demo-1.mvd",
		"players": []map[string]any{
			{"name": "Alpha"},
		},
	})

	testCases := []struct {
		name       string
		payload    json.RawMessage
		wantReason string
		wantDemo   string
	}{
		{
			name:       "missing payload",
			payload:    nil,
			wantReason: SkipReasonMissingStatsPayload,
		},
		{
			name:       "json null payload",
			payload:    json.RawMessage("null"),
			wantReason: SkipReasonMissingStatsPayload,
		},
		{
			name:       "invalid structure",
			payload:    mustJSON(t, map[string]any{"demo": "demo-2.mvd"}),
			wantReason: SkipReasonInvalidStatsPayload,
		},
		{
			name:       "valid payload",
			payload:    validPayload,
			wantReason: "",
			wantDemo:   "demo-1.mvd",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var stats model.SourceStatsMatch
			reason := classifySkipReason(tc.payload, &stats)
			if reason != tc.wantReason {
				t.Fatalf("classifySkipReason() = %q, want %q", reason, tc.wantReason)
			}
			if tc.wantDemo != "" && stats.Demo != tc.wantDemo {
				t.Fatalf("stats.Demo = %q, want %q", stats.Demo, tc.wantDemo)
			}
		})
	}
}

func TestListMatchesForConsolidationRequiresPositiveLimit(t *testing.T) {
	t.Parallel()

	source := &PostgresSource{}
	_, _, err := source.ListMatchesForConsolidation(context.Background(), model.Cursor{}, 0)
	if err == nil || !strings.Contains(err.Error(), "source is not initialized") {
		t.Fatalf("ListMatchesForConsolidation() error = %v, want source is not initialized", err)
	}
}

func TestListMatchesForConsolidationIntegration(t *testing.T) {
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
	if _, err := lockConn.Exec(ctx, "SELECT pg_advisory_lock($1)", integrationDBLockID); err != nil {
		t.Fatalf("pg_advisory_lock() error = %v", err)
	}
	defer func() {
		_, _ = lockConn.Exec(ctx, "SELECT pg_advisory_unlock($1)", integrationDBLockID)
	}()

	resetSourceSchema(t, ctx, pool)
	applyCollectorSchema(t, ctx, pool)

	firstID := insertCollectorMatch(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-missing.mvd",
		mapName:      "dm6",
		mode:         "team",
		playedAt:     time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
		statsPayload: nil,
	})
	secondID := insertCollectorMatch(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-valid.mvd",
		mapName:      "aerowalk",
		mode:         "duel",
		playedAt:     time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
		statsPayload: mustJSON(t, map[string]any{"demo": "demo-valid.mvd", "players": []map[string]any{{"name": "Alpha", "team": "red"}}}),
	})
	thirdID := insertCollectorMatch(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-2",
		demoName:     "demo-invalid.mvd",
		mapName:      "dm4",
		mode:         "team",
		playedAt:     time.Date(2026, 3, 26, 13, 0, 0, 0, time.UTC),
		statsPayload: mustJSON(t, map[string]any{"demo": "", "players": []map[string]any{{"name": "Bravo"}}}),
	})

	source := NewPostgresSource(pool)

	pageOne, cursorOne, err := source.ListMatchesForConsolidation(ctx, model.Cursor{}, 2)
	if err != nil {
		t.Fatalf("ListMatchesForConsolidation() page one error = %v", err)
	}
	if len(pageOne) != 2 {
		t.Fatalf("page one len = %d, want 2", len(pageOne))
	}
	if pageOne[0].CollectorMatchID != firstID || pageOne[1].CollectorMatchID != secondID {
		t.Fatalf("page one ids = [%d %d], want [%d %d]", pageOne[0].CollectorMatchID, pageOne[1].CollectorMatchID, firstID, secondID)
	}
	if pageOne[0].SkipReason != SkipReasonMissingStatsPayload {
		t.Fatalf("page one first skip reason = %q, want %q", pageOne[0].SkipReason, SkipReasonMissingStatsPayload)
	}
	if !pageOne[1].Eligible() || pageOne[1].Stats.Demo != "demo-valid.mvd" {
		t.Fatalf("page one second match = %#v, want eligible parsed stats", pageOne[1])
	}
	if cursorOne.LastCollectorMatchID != secondID {
		t.Fatalf("cursor one last_collector_match_id = %d, want %d", cursorOne.LastCollectorMatchID, secondID)
	}

	pageTwo, cursorTwo, err := source.ListMatchesForConsolidation(ctx, cursorOne, 2)
	if err != nil {
		t.Fatalf("ListMatchesForConsolidation() page two error = %v", err)
	}
	if len(pageTwo) != 1 {
		t.Fatalf("page two len = %d, want 1", len(pageTwo))
	}
	if pageTwo[0].CollectorMatchID != thirdID {
		t.Fatalf("page two id = %d, want %d", pageTwo[0].CollectorMatchID, thirdID)
	}
	if pageTwo[0].SkipReason != SkipReasonInvalidStatsPayload {
		t.Fatalf("page two skip reason = %q, want %q", pageTwo[0].SkipReason, SkipReasonInvalidStatsPayload)
	}
	if cursorTwo.LastCollectorMatchID != thirdID {
		t.Fatalf("cursor two last_collector_match_id = %d, want %d", cursorTwo.LastCollectorMatchID, thirdID)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

type collectorMatchSeed struct {
	serverKey    string
	demoName     string
	mapName      string
	mode         string
	playedAt     time.Time
	statsPayload json.RawMessage
}

func resetSourceSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS player_match_stats",
		"DROP TABLE IF EXISTS player_aliases",
		"DROP TABLE IF EXISTS player_canonical",
		"DROP TABLE IF EXISTS player_stats_checkpoints",
		"DROP TABLE IF EXISTS player_stats_schema_migrations",
		"DROP TABLE IF EXISTS collector_matches",
		"DROP TABLE IF EXISTS collector_schema_migrations",
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
}

func applyCollectorSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "collector", "internal", "storage", "migrations", "001_create_collector_matches.sql")
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", migrationPath, err)
	}
	if _, err := pool.Exec(ctx, string(data)); err != nil {
		t.Fatalf("apply collector migration: %v", err)
	}
}

func insertCollectorMatch(t *testing.T, ctx context.Context, pool *pgxpool.Pool, seed collectorMatchSeed) int64 {
	t.Helper()

	payload := seed.statsPayload
	if len(payload) == 0 {
		payload = nil
	}

	scorePayload := mustJSON(t, map[string]any{"demo": seed.demoName})

	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO collector_matches (
			server_key,
			server_name,
			demo_name,
			match_key,
			mode,
			map_name,
			participants,
			played_at,
			duration_seconds,
			hostname,
			has_bots,
			score_payload,
			stats_payload,
			merged_payload,
			ingested_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NULL, $7, 600, NULL, false, $8, $9, $8, $7, $7)
		RETURNING id
	`, seed.serverKey, "Qlash", seed.demoName, seed.serverKey+":"+seed.demoName, seed.mode, seed.mapName, seed.playedAt, scorePayload, payload).Scan(&id)
	if err != nil {
		t.Fatalf("insert collector_matches row: %v", err)
	}

	return id
}
