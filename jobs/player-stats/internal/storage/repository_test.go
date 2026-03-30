package storage

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

func TestNullIfEmpty(t *testing.T) {
	t.Parallel()

	if value := nullIfEmpty("   "); value != nil {
		t.Fatalf("nullIfEmpty whitespace = %#v, want nil", value)
	}
	if value := nullIfEmpty("alpha"); value != "alpha" {
		t.Fatalf("nullIfEmpty alpha = %#v, want alpha", value)
	}
}

func TestCoalesceTime(t *testing.T) {
	t.Parallel()

	primary := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)
	fallback := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)

	if value := coalesceTime(primary, fallback); !value.Equal(primary) {
		t.Fatalf("coalesceTime(primary, fallback) = %s, want %s", value, primary)
	}
	if value := coalesceTime(time.Time{}, fallback); !value.Equal(fallback) {
		t.Fatalf("coalesceTime(zero, fallback) = %s, want %s", value, fallback)
	}
}

func TestMigrationSQLContainsExpectedSnippets(t *testing.T) {
	t.Parallel()

	data, err := migrationFiles.ReadFile("migrations/001_create_player_stats_analytics.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	sql := string(data)
	for _, snippet := range []string{
		"CREATE TABLE IF NOT EXISTS player_canonical",
		"player_canonical_primary_login_uidx",
		"CREATE TABLE IF NOT EXISTS player_aliases",
		"player_aliases_alias_login_uidx",
		"CREATE TABLE IF NOT EXISTS player_match_stats",
		"player_match_stats_normalized_mode_check",
		"player_match_stats_collector_player_observed_key",
		"CREATE TABLE IF NOT EXISTS player_stats_checkpoints",
	} {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("migration SQL missing %q", snippet)
		}
	}
}

func TestRepositoryLoadCheckpointReturnsZeroValueWhenMissing(t *testing.T) {
	t.Parallel()

	repository := &Repository{}
	_, err := repository.LoadCheckpoint(context.Background(), "player-stats")
	if err == nil || !strings.Contains(err.Error(), "repository is not initialized") {
		t.Fatalf("LoadCheckpoint() error = %v, want repository is not initialized", err)
	}
}

func TestRepositoryUpsertBatchRequiresCheckpointJobName(t *testing.T) {
	t.Parallel()

	repository := &Repository{}
	_, err := repository.UpsertBatch(context.Background(), model.ConsolidationBatch{})
	if err == nil || !strings.Contains(err.Error(), "repository is not initialized") {
		t.Fatalf("UpsertBatch() error = %v, want repository is not initialized", err)
	}
}

func TestRepositoryResolvePlayerRequiresRepository(t *testing.T) {
	t.Parallel()

	repository := &Repository{}
	_, err := repository.ResolvePlayer(context.Background(), model.ResolvePlayerInput{
		ObservedName: "Alpha",
	})
	if err == nil || !strings.Contains(err.Error(), "repository is not initialized") {
		t.Fatalf("ResolvePlayer() error = %v, want repository is not initialized", err)
	}
}

func TestRepositoryUpsertBatchIntegration(t *testing.T) {
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

	repository := NewRepository(pool)
	repository.now = func() time.Time {
		return time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	}

	resetAnalyticsSchema(t, ctx, pool)
	applyCollectorSchema(t, ctx, pool)
	if err := repository.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	matchOneID := insertCollectorMatch(t, ctx, pool, collectorMatchSeed{
		serverKey: "qlash-br-1",
		demoName:  "demo-1.mvd",
		mapName:   "dm6",
		playedAt:  time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
	})
	matchTwoID := insertCollectorMatch(t, ctx, pool, collectorMatchSeed{
		serverKey: "qlash-br-1",
		demoName:  "demo-2.mvd",
		mapName:   "aerowalk",
		playedAt:  time.Date(2026, 3, 26, 13, 0, 0, 0, time.UTC),
	})

	firstBatch := model.ConsolidationBatch{
		Rows: []model.PlayerMatchRow{
			testPlayerMatchRow(t, matchOneID, "demo-1.mvd", "Alpha", "alpha-login", 32),
		},
		Checkpoint: model.Checkpoint{
			JobName:              "player-stats",
			LastCollectorMatchID: matchOneID,
		},
	}

	firstResult, err := repository.UpsertBatch(ctx, firstBatch)
	if err != nil {
		t.Fatalf("UpsertBatch() first error = %v", err)
	}
	if firstResult.CanonicalInserted != 1 || firstResult.AliasesInserted != 1 || firstResult.StatsInserted != 1 {
		t.Fatalf("first UpsertBatch() result = %#v", firstResult)
	}

	secondBatch := model.ConsolidationBatch{
		Rows: []model.PlayerMatchRow{
			testPlayerMatchRow(t, matchOneID, "demo-1.mvd", "Alpha", "alpha-login", 40),
			testPlayerMatchRow(t, matchTwoID, "demo-2.mvd", "AlphaRenamed", "alpha-login", 28),
		},
		Checkpoint: model.Checkpoint{
			JobName:              "player-stats",
			LastCollectorMatchID: matchTwoID,
		},
	}

	secondResult, err := repository.UpsertBatch(ctx, secondBatch)
	if err != nil {
		t.Fatalf("UpsertBatch() second error = %v", err)
	}
	if secondResult.CanonicalInserted != 0 || secondResult.CanonicalReused != 2 {
		t.Fatalf("second UpsertBatch() identity result = %#v", secondResult)
	}
	if secondResult.AliasesInserted != 1 || secondResult.AliasesUpdated != 1 {
		t.Fatalf("second UpsertBatch() alias result = %#v", secondResult)
	}
	if secondResult.StatsInserted != 1 || secondResult.StatsUpdated != 1 {
		t.Fatalf("second UpsertBatch() stats result = %#v", secondResult)
	}

	reprocessResult, err := repository.UpsertBatch(ctx, secondBatch)
	if err != nil {
		t.Fatalf("UpsertBatch() reprocess error = %v", err)
	}
	if reprocessResult.CanonicalReused != 2 || reprocessResult.AliasesUpdated != 2 || reprocessResult.StatsUpdated != 2 {
		t.Fatalf("reprocess UpsertBatch() result = %#v", reprocessResult)
	}

	checkpoint, err := repository.LoadCheckpoint(ctx, "player-stats")
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if checkpoint.LastCollectorMatchID != matchTwoID {
		t.Fatalf("LastCollectorMatchID = %d, want %d", checkpoint.LastCollectorMatchID, matchTwoID)
	}

	var canonicalCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_canonical").Scan(&canonicalCount); err != nil {
		t.Fatalf("count canonical: %v", err)
	}
	if canonicalCount != 1 {
		t.Fatalf("canonical count = %d, want 1", canonicalCount)
	}

	var aliasCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_aliases").Scan(&aliasCount); err != nil {
		t.Fatalf("count aliases: %v", err)
	}
	if aliasCount != 2 {
		t.Fatalf("alias count = %d, want 2", aliasCount)
	}

	var statsCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_match_stats").Scan(&statsCount); err != nil {
		t.Fatalf("count player_match_stats: %v", err)
	}
	if statsCount != 2 {
		t.Fatalf("player_match_stats count = %d, want 2", statsCount)
	}

	var frags int
	if err := pool.QueryRow(ctx, `
		SELECT frags
		FROM player_match_stats
		WHERE collector_match_id = $1
		  AND observed_name = $2
	`, matchOneID, "Alpha").Scan(&frags); err != nil {
		t.Fatalf("select frags: %v", err)
	}
	if frags != 40 {
		t.Fatalf("frags = %d, want 40", frags)
	}
}

func TestRepositoryResolvePlayerIntegration(t *testing.T) {
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

	repository := NewRepository(pool)
	resetAnalyticsSchema(t, ctx, pool)
	applyCollectorSchema(t, ctx, pool)
	if err := repository.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	createdAt := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)
	createdIdentity, err := repository.ResolvePlayer(ctx, model.ResolvePlayerInput{
		ObservedName: "Alpha",
		ObservedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("ResolvePlayer() create error = %v", err)
	}
	if createdIdentity.Resolution != "created" {
		t.Fatalf("created resolution = %q, want created", createdIdentity.Resolution)
	}

	reusedByAlias, err := repository.ResolvePlayer(ctx, model.ResolvePlayerInput{
		ObservedName: "Alpha",
		ObservedAt:   createdAt.Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("ResolvePlayer() alias reuse error = %v", err)
	}
	if reusedByAlias.PlayerID != createdIdentity.PlayerID {
		t.Fatalf("alias reuse player_id = %q, want %q", reusedByAlias.PlayerID, createdIdentity.PlayerID)
	}
	if reusedByAlias.Resolution != "reused_alias" {
		t.Fatalf("alias reuse resolution = %q, want reused_alias", reusedByAlias.Resolution)
	}

	reusedByPromotedAlias, err := repository.ResolvePlayer(ctx, model.ResolvePlayerInput{
		ObservedName:  "Alpha",
		ObservedLogin: "alpha-login",
		ObservedAt:    createdAt.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("ResolvePlayer() promoted alias reuse error = %v", err)
	}
	if reusedByPromotedAlias.PlayerID != createdIdentity.PlayerID {
		t.Fatalf("promoted alias reuse player_id = %q, want %q", reusedByPromotedAlias.PlayerID, createdIdentity.PlayerID)
	}
	if reusedByPromotedAlias.Resolution != "reused_alias" {
		t.Fatalf("promoted alias reuse resolution = %q, want reused_alias", reusedByPromotedAlias.Resolution)
	}

	reusedByLogin, err := repository.ResolvePlayer(ctx, model.ResolvePlayerInput{
		ObservedName:  "AlphaRenamed",
		ObservedLogin: "alpha-login",
		ObservedAt:    createdAt.Add(4 * time.Hour),
	})
	if err != nil {
		t.Fatalf("ResolvePlayer() login reuse error = %v", err)
	}
	if reusedByLogin.PlayerID != createdIdentity.PlayerID {
		t.Fatalf("login reuse player_id = %q, want %q", reusedByLogin.PlayerID, createdIdentity.PlayerID)
	}
	if reusedByLogin.Resolution != "reused_login" {
		t.Fatalf("login reuse resolution = %q, want reused_login", reusedByLogin.Resolution)
	}

	var primaryLogin string
	if err := pool.QueryRow(ctx, "SELECT primary_login FROM player_canonical WHERE id = $1::uuid", createdIdentity.PlayerID).Scan(&primaryLogin); err != nil {
		t.Fatalf("select primary_login: %v", err)
	}
	if primaryLogin != "alpha-login" {
		t.Fatalf("primary_login = %q, want alpha-login", primaryLogin)
	}

	var aliasCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM player_aliases").Scan(&aliasCount); err != nil {
		t.Fatalf("count aliases: %v", err)
	}
	if aliasCount != 3 {
		t.Fatalf("alias count = %d, want 3", aliasCount)
	}

	var firstSeen time.Time
	var lastSeen time.Time
	if err := pool.QueryRow(ctx, `
		SELECT first_seen_at, last_seen_at
		FROM player_aliases
		WHERE alias_name = $1
		  AND COALESCE(login, '') = ''
	`, "Alpha").Scan(&firstSeen, &lastSeen); err != nil {
		t.Fatalf("select alias timestamps: %v", err)
	}
	if !firstSeen.Equal(createdAt) {
		t.Fatalf("first_seen_at = %s, want %s", firstSeen, createdAt)
	}
	if !lastSeen.Equal(createdAt.Add(1 * time.Hour)) {
		t.Fatalf("last_seen_at = %s, want %s", lastSeen, createdAt.Add(1*time.Hour))
	}

	if _, err := repository.ResolvePlayer(ctx, model.ResolvePlayerInput{
		ObservedName: "Alpha",
		ObservedAt:   createdAt.Add(5 * time.Hour),
	}); err != nil {
		t.Fatalf("ResolvePlayer() alias window update error = %v", err)
	}

	if err := pool.QueryRow(ctx, `
		SELECT first_seen_at, last_seen_at
		FROM player_aliases
		WHERE alias_name = $1
		  AND COALESCE(login, '') = ''
	`, "Alpha").Scan(&firstSeen, &lastSeen); err != nil {
		t.Fatalf("select alias timestamps after update: %v", err)
	}
	if !firstSeen.Equal(createdAt) {
		t.Fatalf("first_seen_at after update = %s, want %s", firstSeen, createdAt)
	}
	if !lastSeen.Equal(createdAt.Add(5 * time.Hour)) {
		t.Fatalf("last_seen_at after update = %s, want %s", lastSeen, createdAt.Add(5*time.Hour))
	}
}

func resetAnalyticsSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	statements := []string{
		"DROP TABLE IF EXISTS player_match_stats",
		"DROP TABLE IF EXISTS player_aliases",
		"DROP TABLE IF EXISTS player_canonical",
		"DROP TABLE IF EXISTS player_stats_checkpoints",
		"DROP TABLE IF EXISTS player_stats_schema_migrations",
		"DROP TABLE IF EXISTS collector_matches",
		"DROP TABLE IF EXISTS collector_schema_migrations",
	}
	for _, statement := range statements {
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

type collectorMatchSeed struct {
	serverKey string
	demoName  string
	mapName   string
	playedAt  time.Time
}

func insertCollectorMatch(t *testing.T, ctx context.Context, pool *pgxpool.Pool, seed collectorMatchSeed) int64 {
	t.Helper()

	payload, err := json.Marshal(map[string]any{"demo": seed.demoName})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var id int64
	err = pool.QueryRow(ctx, `
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
		VALUES ($1, $2, $3, $4, $5, $6, NULL, $7, 600, NULL, false, $8, $8, $8, $7, $7)
		RETURNING id
	`, seed.serverKey, "Qlash", seed.demoName, seed.serverKey+":"+seed.demoName, "2on2", seed.mapName, seed.playedAt, payload).Scan(&id)
	if err != nil {
		t.Fatalf("insert collector_matches row: %v", err)
	}

	return id
}

func testPlayerMatchRow(t *testing.T, collectorMatchID int64, demoName string, observedName string, login string, frags int) model.PlayerMatchRow {
	t.Helper()

	snapshot, err := json.Marshal(map[string]any{
		"name":  observedName,
		"login": login,
		"frags": frags,
	})
	if err != nil {
		t.Fatalf("json.Marshal() snapshot error = %v", err)
	}

	return model.PlayerMatchRow{
		CollectorMatchID:      collectorMatchID,
		ServerKey:             "qlash-br-1",
		DemoName:              demoName,
		ObservedName:          observedName,
		ObservedLogin:         login,
		Team:                  "red",
		MapName:               "dm6",
		RawMode:               "team",
		NormalizedMode:        "2on2",
		PlayedAt:              time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
		HasBots:               false,
		ExcludedFromAnalytics: false,
		Frags:                 frags,
		Deaths:                12,
		Kills:                 30,
		TeamKills:             1,
		Suicides:              0,
		DamageTaken:           2500,
		DamageGiven:           3200,
		SpreeMax:              4,
		SpreeQuad:             1,
		RLHits:                18,
		RLKills:               10,
		LGAttacks:             100,
		LGHits:                42,
		GA:                    2,
		RA:                    3,
		YA:                    5,
		Health100:             6,
		Ping:                  22,
		Efficiency:            71.43,
		LGAccuracy:            42,
		StatsSnapshot:         snapshot,
		ConsolidatedAt:        time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC),
	}
}
