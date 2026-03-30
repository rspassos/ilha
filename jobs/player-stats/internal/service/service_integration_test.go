package service

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rspassos/ilha/jobs/player-stats/internal/logging"
	"github.com/rspassos/ilha/jobs/player-stats/internal/metrics"
	"github.com/rspassos/ilha/jobs/player-stats/internal/source"
	"github.com/rspassos/ilha/jobs/player-stats/internal/storage"
)

const integrationDBLockID int64 = 505100

func TestRunOnceIntegrationProcessesCompleteBatch(t *testing.T) {
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

	resetServiceSchema(t, ctx, pool)
	applyCollectorSchemaForService(t, ctx, pool)

	repository := storage.NewRepository(pool)
	if err := repository.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	firstID := insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-success-1.mvd",
		mapName:      "dm6",
		mode:         "team",
		playedAt:     time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
		hasBots:      false,
		statsPayload: validStatsPayload(t, "demo-success-1.mvd", false),
	})
	insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-skip.mvd",
		mapName:      "dm6",
		mode:         "team",
		playedAt:     time.Date(2026, 3, 26, 12, 30, 0, 0, time.UTC),
		hasBots:      false,
		statsPayload: nil,
	})
	thirdID := insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-success-2.mvd",
		mapName:      "dm4",
		mode:         "team",
		playedAt:     time.Date(2026, 3, 26, 13, 0, 0, 0, time.UTC),
		hasBots:      true,
		statsPayload: validStatsPayload(t, "demo-success-2.mvd", true),
	})

	service := New(
		logging.New(&bytes.Buffer{}, "player-stats"),
		metrics.New(),
		repository,
		source.NewPostgresSource(pool),
		"player-stats",
		3,
	)

	if err := service.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	checkpoint, err := repository.LoadCheckpoint(ctx, "player-stats")
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if checkpoint.LastCollectorMatchID != thirdID {
		t.Fatalf("checkpoint.LastCollectorMatchID = %d, want %d", checkpoint.LastCollectorMatchID, thirdID)
	}

	var statsCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats").Scan(&statsCount); err != nil {
		t.Fatalf("count player_match_stats: %v", err)
	}
	if statsCount != 8 {
		t.Fatalf("player_match_stats count = %d, want 8", statsCount)
	}

	var excludedCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats WHERE excluded_from_analytics = true").Scan(&excludedCount); err != nil {
		t.Fatalf("count excluded rows: %v", err)
	}
	if excludedCount != 4 {
		t.Fatalf("excluded rows count = %d, want 4", excludedCount)
	}

	var firstMatchRows int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats WHERE collector_match_id = $1", firstID).Scan(&firstMatchRows); err != nil {
		t.Fatalf("count first match rows: %v", err)
	}
	if firstMatchRows != 4 {
		t.Fatalf("first match rows = %d, want 4", firstMatchRows)
	}
}

func TestRunOnceIntegrationRetriesFailedMatchIdempotently(t *testing.T) {
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

	resetServiceSchema(t, ctx, pool)
	applyCollectorSchemaForService(t, ctx, pool)

	repository := storage.NewRepository(pool)
	if err := repository.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	firstID := insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-retry-1.mvd",
		mapName:      "dm6",
		mode:         "team",
		playedAt:     time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
		statsPayload: validStatsPayload(t, "demo-retry-1.mvd", false),
	})
	secondID := insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-retry-2.mvd",
		mapName:      "dm6",
		mode:         "team",
		playedAt:     time.Date(2026, 3, 26, 12, 30, 0, 0, time.UTC),
		statsPayload: invalidPlayerCountPayload(t, "demo-retry-2.mvd"),
	})
	thirdID := insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-retry-3.mvd",
		mapName:      "dm4",
		mode:         "team",
		playedAt:     time.Date(2026, 3, 26, 13, 0, 0, 0, time.UTC),
		statsPayload: validStatsPayload(t, "demo-retry-3.mvd", false),
	})

	service := New(
		logging.New(&bytes.Buffer{}, "player-stats"),
		metrics.New(),
		repository,
		source.NewPostgresSource(pool),
		"player-stats",
		3,
	)

	if err := service.RunOnce(ctx); err == nil || !strings.Contains(err.Error(), "cycle stopped after batch failures") {
		t.Fatalf("first RunOnce() error = %v, want batch failure", err)
	}

	checkpoint, err := repository.LoadCheckpoint(ctx, "player-stats")
	if err != nil {
		t.Fatalf("LoadCheckpoint() after failure error = %v", err)
	}
	if checkpoint.LastCollectorMatchID != firstID {
		t.Fatalf("checkpoint after failure = %d, want %d", checkpoint.LastCollectorMatchID, firstID)
	}

	var countAfterFailure int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats").Scan(&countAfterFailure); err != nil {
		t.Fatalf("count after failure: %v", err)
	}
	if countAfterFailure != 8 {
		t.Fatalf("rows after failure = %d, want 8", countAfterFailure)
	}

	if _, err := pool.Exec(ctx, "UPDATE collector_matches SET stats_payload = $1, updated_at = now() WHERE id = $2", validStatsPayload(t, "demo-retry-2.mvd", false), secondID); err != nil {
		t.Fatalf("update failing match payload: %v", err)
	}

	if err := service.RunOnce(ctx); err != nil {
		t.Fatalf("second RunOnce() error = %v", err)
	}

	checkpoint, err = repository.LoadCheckpoint(ctx, "player-stats")
	if err != nil {
		t.Fatalf("LoadCheckpoint() after retry error = %v", err)
	}
	if checkpoint.LastCollectorMatchID != thirdID {
		t.Fatalf("checkpoint after retry = %d, want %d", checkpoint.LastCollectorMatchID, thirdID)
	}

	var finalCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats").Scan(&finalCount); err != nil {
		t.Fatalf("final player_match_stats count: %v", err)
	}
	if finalCount != 12 {
		t.Fatalf("final player_match_stats count = %d, want 12", finalCount)
	}

	var secondMatchRows int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats WHERE collector_match_id = $1", secondID).Scan(&secondMatchRows); err != nil {
		t.Fatalf("count second match rows: %v", err)
	}
	if secondMatchRows != 4 {
		t.Fatalf("second match rows = %d, want 4", secondMatchRows)
	}
}

func TestRunOnceIntegrationPreservesAliasesBotsAndIdempotency(t *testing.T) {
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

	resetServiceSchema(t, ctx, pool)
	applyCollectorSchemaForService(t, ctx, pool)

	repository := storage.NewRepository(pool)
	if err := repository.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-alias-1.mvd",
		mapName:      "aerowalk",
		mode:         "duel",
		playedAt:     time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC),
		hasBots:      false,
		statsPayload: duelStatsPayload(t, "demo-alias-1.mvd", duelPlayerSeed{name: "Alpha", login: "alpha-login"}, duelPlayerSeed{name: "Bravo", login: "bravo-login"}),
	})
	insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-alias-2.mvd",
		mapName:      "aerowalk",
		mode:         "duel",
		playedAt:     time.Date(2026, 3, 26, 14, 10, 0, 0, time.UTC),
		hasBots:      false,
		statsPayload: duelStatsPayload(t, "demo-alias-2.mvd", duelPlayerSeed{name: "AlphaRenamed", login: "alpha-login"}, duelPlayerSeed{name: "Bravo", login: "bravo-login"}),
	})
	thirdID := insertCollectorMatchForService(t, ctx, pool, collectorMatchSeed{
		serverKey:    "qlash-br-1",
		demoName:     "demo-bot-1.mvd",
		mapName:      "dm6",
		mode:         "duel",
		playedAt:     time.Date(2026, 3, 26, 14, 20, 0, 0, time.UTC),
		hasBots:      true,
		statsPayload: duelStatsPayload(t, "demo-bot-1.mvd", duelPlayerSeed{name: "AlphaRenamed", login: "alpha-login"}, duelPlayerSeed{name: "BotDelta", login: "", bot: true}),
	})

	service := New(
		logging.New(&bytes.Buffer{}, "player-stats"),
		metrics.New(),
		repository,
		source.NewPostgresSource(pool),
		"player-stats",
		10,
	)

	if err := service.RunOnce(ctx); err != nil {
		t.Fatalf("first RunOnce() error = %v", err)
	}
	if err := service.RunOnce(ctx); err != nil {
		t.Fatalf("second RunOnce() error = %v", err)
	}

	checkpoint, err := repository.LoadCheckpoint(ctx, "player-stats")
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if checkpoint.LastCollectorMatchID != thirdID {
		t.Fatalf("checkpoint.LastCollectorMatchID = %d, want %d", checkpoint.LastCollectorMatchID, thirdID)
	}

	var statsCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats").Scan(&statsCount); err != nil {
		t.Fatalf("count player_match_stats: %v", err)
	}
	if statsCount != 6 {
		t.Fatalf("player_match_stats count = %d, want 6", statsCount)
	}

	var excludedCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats WHERE excluded_from_analytics = true").Scan(&excludedCount); err != nil {
		t.Fatalf("count excluded rows: %v", err)
	}
	if excludedCount != 2 {
		t.Fatalf("excluded rows count = %d, want 2", excludedCount)
	}

	var canonicalCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_canonical").Scan(&canonicalCount); err != nil {
		t.Fatalf("count player_canonical: %v", err)
	}
	if canonicalCount != 3 {
		t.Fatalf("player_canonical count = %d, want 3", canonicalCount)
	}

	var aliasCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_aliases").Scan(&aliasCount); err != nil {
		t.Fatalf("count player_aliases: %v", err)
	}
	if aliasCount != 4 {
		t.Fatalf("player_aliases count = %d, want 4", aliasCount)
	}

	var alphaAliasVariants int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM player_aliases
		WHERE player_id = (
			SELECT player_id
			FROM player_aliases
			WHERE alias_name = 'Alpha'
			  AND COALESCE(login, '') = 'alpha-login'
		)
	`).Scan(&alphaAliasVariants); err != nil {
		t.Fatalf("count alpha alias variants: %v", err)
	}
	if alphaAliasVariants != 2 {
		t.Fatalf("alpha alias variants = %d, want 2", alphaAliasVariants)
	}

	var alphaBotRows int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM player_match_stats
		WHERE observed_name = 'AlphaRenamed'
		  AND excluded_from_analytics = true
	`).Scan(&alphaBotRows); err != nil {
		t.Fatalf("count alpha bot rows: %v", err)
	}
	if alphaBotRows != 1 {
		t.Fatalf("alpha bot rows = %d, want 1", alphaBotRows)
	}
}

type collectorMatchSeed struct {
	serverKey    string
	demoName     string
	mapName      string
	mode         string
	playedAt     time.Time
	hasBots      bool
	statsPayload json.RawMessage
}

type duelPlayerSeed struct {
	name  string
	login string
	bot   bool
}

func resetServiceSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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

func applyCollectorSchemaForService(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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

func insertCollectorMatchForService(t *testing.T, ctx context.Context, pool *pgxpool.Pool, seed collectorMatchSeed) int64 {
	t.Helper()

	payload := seed.statsPayload
	if len(payload) == 0 {
		payload = nil
	}

	scorePayload := mustJSONForService(t, map[string]any{"demo": seed.demoName})

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
		VALUES (
			$1, 'Qlash', $2, $1 || ':' || $2, $3, $4, '[]'::jsonb, $5, 900, 'localhost', $6,
			$7::jsonb, $8::jsonb, '{}'::jsonb, $5, $5
		)
		RETURNING id
	`, seed.serverKey, seed.demoName, seed.mode, seed.mapName, seed.playedAt, seed.hasBots, scorePayload, payload).Scan(&id)
	if err != nil {
		t.Fatalf("insert collector match %s: %v", seed.demoName, err)
	}

	return id
}

func validStatsPayload(t *testing.T, demo string, includeBot bool) json.RawMessage {
	t.Helper()

	players := []map[string]any{
		servicePlayerPayload("Alpha", "alpha-login", "red", false),
		servicePlayerPayload("Bravo", "bravo-login", "blue", false),
		servicePlayerPayload("Charlie", "", "red", false),
		servicePlayerPayload("Delta", "", "blue", includeBot),
	}

	return mustJSONForService(t, map[string]any{
		"demo":    demo,
		"map":     "dm6",
		"mode":    "team",
		"dm":      3,
		"players": players,
	})
}

func duelStatsPayload(t *testing.T, demo string, first duelPlayerSeed, second duelPlayerSeed) json.RawMessage {
	t.Helper()

	return mustJSONForService(t, map[string]any{
		"demo": demo,
		"map":  "aerowalk",
		"mode": "duel",
		"dm":   3,
		"players": []map[string]any{
			servicePlayerPayload(first.name, first.login, "red", first.bot),
			servicePlayerPayload(second.name, second.login, "blue", second.bot),
		},
	})
}

func invalidPlayerCountPayload(t *testing.T, demo string) json.RawMessage {
	t.Helper()

	return mustJSONForService(t, map[string]any{
		"demo": demo,
		"map":  "dm6",
		"mode": "team",
		"dm":   3,
		"players": []map[string]any{
			servicePlayerPayload("Alpha", "alpha-login", "red", false),
			servicePlayerPayload("Bravo", "bravo-login", "blue", false),
			servicePlayerPayload("Charlie", "", "red", false),
		},
	})
}

func servicePlayerPayload(name string, login string, team string, bot bool) map[string]any {
	player := map[string]any{
		"name":  name,
		"login": login,
		"team":  team,
		"ping":  30,
		"stats": map[string]any{
			"frags":  20,
			"deaths": 10,
			"kills":  20,
		},
		"dmg": map[string]any{
			"given": 2000,
			"taken": 1800,
		},
		"spree": map[string]any{
			"max":  3,
			"quad": 0,
		},
		"weapons": map[string]any{
			"lg": map[string]any{
				"acc": map[string]any{"attacks": 100, "hits": 40},
			},
			"rl": map[string]any{
				"acc":   map[string]any{"hits": 12},
				"kills": map[string]any{"total": 8},
			},
		},
		"items": map[string]any{
			"ga":         map[string]any{"took": 1},
			"ra":         map[string]any{"took": 2},
			"ya":         map[string]any{"took": 3},
			"health_100": map[string]any{"took": 4},
		},
		"bot": map[string]any{
			"skill":      0,
			"customised": false,
		},
	}
	if bot {
		player["bot"] = map[string]any{
			"skill":      7,
			"customised": true,
		}
	}
	return player
}

func mustJSONForService(t *testing.T, value any) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}
