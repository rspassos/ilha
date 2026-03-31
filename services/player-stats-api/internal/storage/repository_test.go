package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rspassos/ilha/services/player-stats-api/internal/model"
)

const integrationDBLockID int64 = 505100

func TestBuildOrderByClause(t *testing.T) {
	t.Parallel()

	orderBy, err := buildOrderByClause("efficiency", "desc")
	if err != nil {
		t.Fatalf("buildOrderByClause() error = %v", err)
	}

	want := "efficiency DESC, frags DESC, matches DESC, player_id ASC"
	if orderBy != want {
		t.Fatalf("orderBy = %q, want %q", orderBy, want)
	}
}

func TestBuildOrderByClauseRejectsUnsupportedSortBy(t *testing.T) {
	t.Parallel()

	if _, err := buildOrderByClause("kills", "desc"); err == nil {
		t.Fatal("buildOrderByClause() error = nil, want non-nil")
	}
}

func TestBuildOrderByClauseRejectsUnsupportedSortDirection(t *testing.T) {
	t.Parallel()

	if _, err := buildOrderByClause("efficiency", "sideways"); err == nil {
		t.Fatal("buildOrderByClause() error = nil, want non-nil")
	}
}

func TestRepositoryListPlayerRankingRequiresRepository(t *testing.T) {
	t.Parallel()

	repository := &Repository{}
	_, err := repository.ListPlayerRanking(context.Background(), model.RankingQuery{
		Limit:          10,
		MinimumMatches: 10,
	})
	if err == nil || !strings.Contains(err.Error(), "repository is not initialized") {
		t.Fatalf("ListPlayerRanking() error = %v, want repository is not initialized", err)
	}
}

func TestRepositoryListPlayerRankingIntegrationWithoutFilters(t *testing.T) {
	repository, pool, ctx := openIntegrationRepository(t)
	defer pool.Close()

	alphaID := insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000010", "Alpha")
	betaID := insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000020", "Beta")
	gammaID := insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000030", "Gamma")
	lowVolumeID := insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000040", "LowVolume")
	excludedID := insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000050", "Excluded")

	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         alphaID,
		PlayerName:       "Alpha",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       12,
		FragsPerMatch:    10,
		KillsPerMatch:    9,
		DeathsPerMatch:   6,
		RLHitsPerMatch:   5,
		Efficiency:       60,
		LGAccuracy:       40,
		StartPlayedAt:    time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "alpha",
		ObservedName:     "Alpha",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         betaID,
		PlayerName:       "Beta",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       12,
		FragsPerMatch:    10,
		KillsPerMatch:    8,
		DeathsPerMatch:   6,
		RLHitsPerMatch:   4,
		Efficiency:       60,
		LGAccuracy:       40,
		StartPlayedAt:    time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "beta",
		ObservedName:     "Beta",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         gammaID,
		PlayerName:       "Gamma",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       12,
		FragsPerMatch:    9,
		KillsPerMatch:    7,
		DeathsPerMatch:   6,
		RLHitsPerMatch:   3,
		Efficiency:       60,
		LGAccuracy:       39,
		StartPlayedAt:    time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "gamma",
		ObservedName:     "Gamma",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         lowVolumeID,
		PlayerName:       "LowVolume",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       9,
		FragsPerMatch:    30,
		KillsPerMatch:    20,
		DeathsPerMatch:   2,
		RLHitsPerMatch:   10,
		Efficiency:       90,
		LGAccuracy:       55,
		StartPlayedAt:    time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "low",
		ObservedName:     "LowVolume",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         excludedID,
		PlayerName:       "Excluded",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       12,
		FragsPerMatch:    50,
		KillsPerMatch:    40,
		DeathsPerMatch:   1,
		RLHitsPerMatch:   20,
		Efficiency:       99,
		LGAccuracy:       60,
		StartPlayedAt:    time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "excluded",
		ObservedName:     "Excluded",
		ExcludedFromRank: true,
	})

	page, err := repository.ListPlayerRanking(ctx, model.RankingQuery{
		Limit:          10,
		MinimumMatches: 10,
		SortBy:         "efficiency",
		SortDirection:  "desc",
	})
	if err != nil {
		t.Fatalf("ListPlayerRanking() error = %v", err)
	}

	if page.Meta.Returned != 3 {
		t.Fatalf("returned = %d, want 3", page.Meta.Returned)
	}
	if page.Meta.HasNext {
		t.Fatal("has_next = true, want false")
	}

	gotNames := []string{
		page.Data[0].DisplayName,
		page.Data[1].DisplayName,
		page.Data[2].DisplayName,
	}
	wantNames := []string{"Alpha", "Beta", "Gamma"}
	for index := range wantNames {
		if gotNames[index] != wantNames[index] {
			t.Fatalf("data[%d].display_name = %q, want %q", index, gotNames[index], wantNames[index])
		}
	}

	if page.Data[0].Rank != 1 || page.Data[1].Rank != 2 || page.Data[2].Rank != 3 {
		t.Fatalf("ranks = [%d %d %d], want [1 2 3]", page.Data[0].Rank, page.Data[1].Rank, page.Data[2].Rank)
	}
	if page.Data[0].Frags != 120 {
		t.Fatalf("Alpha frags = %d, want 120", page.Data[0].Frags)
	}
}

func TestRepositoryListPlayerRankingIntegrationWithCombinedFilters(t *testing.T) {
	repository, pool, ctx := openIntegrationRepository(t)
	defer pool.Close()

	filteredID := insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000060", "Filtered")
	otherID := insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000070", "Other")
	excludedID := insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000080", "ExcludedFilter")

	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         filteredID,
		PlayerName:       "Filtered",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       10,
		FragsPerMatch:    11,
		KillsPerMatch:    9,
		DeathsPerMatch:   7,
		RLHitsPerMatch:   4,
		Efficiency:       58.25,
		LGAccuracy:       43.5,
		StartPlayedAt:    time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "filtered-ok",
		ObservedName:     "Filtered",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         filteredID,
		PlayerName:       "Filtered",
		ServerKey:        "server-b",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       2,
		FragsPerMatch:    50,
		KillsPerMatch:    45,
		DeathsPerMatch:   1,
		RLHitsPerMatch:   20,
		Efficiency:       99,
		LGAccuracy:       70,
		StartPlayedAt:    time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "filtered-server-b",
		ObservedName:     "Filtered",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         filteredID,
		PlayerName:       "Filtered",
		ServerKey:        "server-a",
		MapName:          "ztndm3",
		Mode:             "2on2",
		MatchCount:       2,
		FragsPerMatch:    50,
		KillsPerMatch:    45,
		DeathsPerMatch:   1,
		RLHitsPerMatch:   20,
		Efficiency:       99,
		LGAccuracy:       70,
		StartPlayedAt:    time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "filtered-map",
		ObservedName:     "Filtered",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         filteredID,
		PlayerName:       "Filtered",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "1on1",
		MatchCount:       2,
		FragsPerMatch:    50,
		KillsPerMatch:    45,
		DeathsPerMatch:   1,
		RLHitsPerMatch:   20,
		Efficiency:       99,
		LGAccuracy:       70,
		StartPlayedAt:    time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "filtered-mode",
		ObservedName:     "Filtered",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         filteredID,
		PlayerName:       "Filtered",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       1,
		FragsPerMatch:    99,
		KillsPerMatch:    99,
		DeathsPerMatch:   1,
		RLHitsPerMatch:   30,
		Efficiency:       99,
		LGAccuracy:       70,
		StartPlayedAt:    time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "filtered-before-range",
		ObservedName:     "Filtered",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         otherID,
		PlayerName:       "Other",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       9,
		FragsPerMatch:    30,
		KillsPerMatch:    25,
		DeathsPerMatch:   4,
		RLHitsPerMatch:   8,
		Efficiency:       80,
		LGAccuracy:       50,
		StartPlayedAt:    time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "other",
		ObservedName:     "Other",
		ExcludedFromRank: false,
	})
	insertMatchSeries(t, ctx, pool, matchSeriesSeed{
		PlayerID:         excludedID,
		PlayerName:       "ExcludedFilter",
		ServerKey:        "server-a",
		MapName:          "dm6",
		Mode:             "2on2",
		MatchCount:       12,
		FragsPerMatch:    50,
		KillsPerMatch:    45,
		DeathsPerMatch:   2,
		RLHitsPerMatch:   9,
		Efficiency:       95,
		LGAccuracy:       55,
		StartPlayedAt:    time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC),
		CollectorPrefix:  "excluded-filter",
		ObservedName:     "ExcludedFilter",
		ExcludedFromRank: true,
	})

	page, err := repository.ListPlayerRanking(ctx, model.RankingQuery{
		Mode:           "2on2",
		Map:            "dm6",
		Server:         "server-a",
		From:           time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		To:             time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
		SortBy:         "frags",
		SortDirection:  "desc",
		Limit:          10,
		Offset:         0,
		MinimumMatches: 10,
	})
	if err != nil {
		t.Fatalf("ListPlayerRanking() error = %v", err)
	}

	if len(page.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(page.Data))
	}

	row := page.Data[0]
	if row.DisplayName != "Filtered" {
		t.Fatalf("display_name = %q, want Filtered", row.DisplayName)
	}
	if row.Matches != 10 {
		t.Fatalf("matches = %d, want 10", row.Matches)
	}
	if row.Frags != 110 {
		t.Fatalf("frags = %d, want 110", row.Frags)
	}
	if row.Kills != 90 {
		t.Fatalf("kills = %d, want 90", row.Kills)
	}
	if row.Deaths != 70 {
		t.Fatalf("deaths = %d, want 70", row.Deaths)
	}
	if row.RLHits != 40 {
		t.Fatalf("rl_hits = %d, want 40", row.RLHits)
	}
	if row.Efficiency != 58.25 {
		t.Fatalf("efficiency = %v, want 58.25", row.Efficiency)
	}
	if row.LGAccuracy != 43.5 {
		t.Fatalf("lg_accuracy = %v, want 43.5", row.LGAccuracy)
	}
}

func TestRepositoryListPlayerRankingIntegrationPaginationAndDeterministicOrder(t *testing.T) {
	repository, pool, ctx := openIntegrationRepository(t)
	defer pool.Close()

	playerIDs := []string{
		insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000090", "Alpha"),
		insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000091", "Bravo"),
		insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000092", "Charlie"),
		insertCanonicalPlayer(t, ctx, pool, "00000000-0000-0000-0000-000000000093", "Delta"),
	}
	names := []string{"Alpha", "Bravo", "Charlie", "Delta"}
	efficiencies := []float64{70, 68, 66, 64}

	for index := range playerIDs {
		insertMatchSeries(t, ctx, pool, matchSeriesSeed{
			PlayerID:         playerIDs[index],
			PlayerName:       names[index],
			ServerKey:        "server-a",
			MapName:          "dm6",
			Mode:             "2on2",
			MatchCount:       10,
			FragsPerMatch:    10 - index,
			KillsPerMatch:    8,
			DeathsPerMatch:   5,
			RLHitsPerMatch:   3,
			Efficiency:       efficiencies[index],
			LGAccuracy:       40,
			StartPlayedAt:    time.Date(2026, 3, 15+index, 12, 0, 0, 0, time.UTC),
			CollectorPrefix:  names[index],
			ObservedName:     names[index],
			ExcludedFromRank: false,
		})
	}

	page, err := repository.ListPlayerRanking(ctx, model.RankingQuery{
		SortBy:         "efficiency",
		SortDirection:  "desc",
		Limit:          2,
		Offset:         1,
		MinimumMatches: 10,
	})
	if err != nil {
		t.Fatalf("ListPlayerRanking() error = %v", err)
	}

	if !page.Meta.HasNext {
		t.Fatal("has_next = false, want true")
	}
	if page.Meta.Returned != 2 {
		t.Fatalf("returned = %d, want 2", page.Meta.Returned)
	}
	if page.Data[0].DisplayName != "Bravo" || page.Data[0].Rank != 2 {
		t.Fatalf("data[0] = %#v, want Bravo with rank 2", page.Data[0])
	}
	if page.Data[1].DisplayName != "Charlie" || page.Data[1].Rank != 3 {
		t.Fatalf("data[1] = %#v, want Charlie with rank 3", page.Data[1])
	}
}

func openIntegrationRepository(t *testing.T) (*Repository, *pgxpool.Pool, context.Context) {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("PLAYER_STATS_API_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("COLLECTOR_TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("PLAYER_STATS_API_TEST_DATABASE_URL or COLLECTOR_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}

	lockConn, err := pool.Acquire(ctx)
	if err != nil {
		pool.Close()
		t.Fatalf("Acquire() error = %v", err)
	}
	if _, err := lockConn.Exec(ctx, "SELECT pg_advisory_lock($1)", integrationDBLockID); err != nil {
		lockConn.Release()
		pool.Close()
		t.Fatalf("pg_advisory_lock() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = lockConn.Exec(ctx, "SELECT pg_advisory_unlock($1)", integrationDBLockID)
		lockConn.Release()
	})

	resetAnalyticsSchema(t, ctx, pool)
	applyCollectorSchema(t, ctx, pool)
	applyPlayerStatsSchema(t, ctx, pool)

	return NewRepository(pool, nil, nil), pool, ctx
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

	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", "jobs", "collector", "internal", "storage", "migrations", "001_create_collector_matches.sql")
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", migrationPath, err)
	}
	if _, err := pool.Exec(ctx, string(data)); err != nil {
		t.Fatalf("apply collector migration: %v", err)
	}
}

func applyPlayerStatsSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", "jobs", "player-stats", "internal", "storage", "migrations", "001_create_player_stats_analytics.sql")
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", migrationPath, err)
	}
	if _, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto"); err != nil {
		t.Fatalf("create pgcrypto extension: %v", err)
	}
	if _, err := pool.Exec(ctx, string(data)); err != nil {
		t.Fatalf("apply player stats migration: %v", err)
	}
}

func insertCanonicalPlayer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string, displayName string) string {
	t.Helper()

	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO player_canonical (id, primary_login, display_name, created_at, updated_at)
		VALUES ($1::uuid, NULL, $2, $3, $3)
	`, id, displayName, now); err != nil {
		t.Fatalf("insert player_canonical: %v", err)
	}

	return id
}

type matchSeriesSeed struct {
	PlayerID         string
	PlayerName       string
	ServerKey        string
	MapName          string
	Mode             string
	MatchCount       int
	FragsPerMatch    int
	KillsPerMatch    int
	DeathsPerMatch   int
	RLHitsPerMatch   int
	Efficiency       float64
	LGAccuracy       float64
	StartPlayedAt    time.Time
	CollectorPrefix  string
	ObservedName     string
	ExcludedFromRank bool
}

func insertMatchSeries(t *testing.T, ctx context.Context, pool *pgxpool.Pool, seed matchSeriesSeed) {
	t.Helper()

	for index := 0; index < seed.MatchCount; index++ {
		playedAt := seed.StartPlayedAt.Add(time.Duration(index) * time.Hour)
		demoName := fmtDemoName(seed.CollectorPrefix, index)
		collectorMatchID := insertCollectorMatch(t, ctx, pool, collectorMatchSeed{
			ServerKey: seed.ServerKey,
			DemoName:  demoName,
			MapName:   seed.MapName,
			PlayedAt:  playedAt,
		})
		insertPlayerMatchStat(t, ctx, pool, playerMatchStatSeed{
			CollectorMatchID:      collectorMatchID,
			PlayerID:              seed.PlayerID,
			ServerKey:             seed.ServerKey,
			DemoName:              demoName,
			ObservedName:          seed.ObservedName,
			MapName:               seed.MapName,
			Mode:                  seed.Mode,
			PlayedAt:              playedAt,
			Frags:                 seed.FragsPerMatch,
			Kills:                 seed.KillsPerMatch,
			Deaths:                seed.DeathsPerMatch,
			RLHits:                seed.RLHitsPerMatch,
			Efficiency:            seed.Efficiency,
			LGAccuracy:            seed.LGAccuracy,
			ExcludedFromAnalytics: seed.ExcludedFromRank,
		})
	}
}

type collectorMatchSeed struct {
	ServerKey string
	DemoName  string
	MapName   string
	PlayedAt  time.Time
}

func insertCollectorMatch(t *testing.T, ctx context.Context, pool *pgxpool.Pool, seed collectorMatchSeed) int64 {
	t.Helper()

	payload, err := json.Marshal(map[string]any{"demo": seed.DemoName})
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
	`, seed.ServerKey, "Qlash", seed.DemoName, seed.ServerKey+":"+seed.DemoName, seed.Mode(), seed.MapName, seed.PlayedAt, payload).Scan(&id)
	if err != nil {
		t.Fatalf("insert collector_matches row: %v", err)
	}

	return id
}

func (s collectorMatchSeed) Mode() string {
	return "2on2"
}

type playerMatchStatSeed struct {
	CollectorMatchID      int64
	PlayerID              string
	ServerKey             string
	DemoName              string
	ObservedName          string
	MapName               string
	Mode                  string
	PlayedAt              time.Time
	Frags                 int
	Kills                 int
	Deaths                int
	RLHits                int
	Efficiency            float64
	LGAccuracy            float64
	ExcludedFromAnalytics bool
}

func insertPlayerMatchStat(t *testing.T, ctx context.Context, pool *pgxpool.Pool, seed playerMatchStatSeed) {
	t.Helper()

	statsSnapshot, err := json.Marshal(map[string]any{
		"name":       seed.ObservedName,
		"demo_name":  seed.DemoName,
		"frags":      seed.Frags,
		"efficiency": seed.Efficiency,
	})
	if err != nil {
		t.Fatalf("json.Marshal() statsSnapshot error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO player_match_stats (
			collector_match_id,
			player_id,
			server_key,
			demo_name,
			observed_name,
			observed_login,
			team,
			map_name,
			raw_mode,
			normalized_mode,
			played_at,
			has_bots,
			excluded_from_analytics,
			frags,
			deaths,
			kills,
			team_kills,
			suicides,
			damage_taken,
			damage_given,
			spree_max,
			spree_quad,
			rl_hits,
			rl_kills,
			lg_attacks,
			lg_hits,
			ga,
			ra,
			ya,
			health_100,
			ping,
			efficiency,
			lg_accuracy,
			stats_snapshot,
			consolidated_at
		)
		VALUES ($1, $2::uuid, $3, $4, $5, NULL, 'red', $6, $7, $8, $9, false, $10, $11, $12, $13, 0, 0, 1000, 1500, 3, 0, $14, 0, 100, 40, 1, 1, 1, 1, 20, $15, $16, $17, $18)
	`, seed.CollectorMatchID, seed.PlayerID, seed.ServerKey, seed.DemoName, seed.ObservedName, seed.MapName, seed.Mode, seed.Mode, seed.PlayedAt, seed.ExcludedFromAnalytics, seed.Frags, seed.Deaths, seed.Kills, seed.RLHits, seed.Efficiency, seed.LGAccuracy, statsSnapshot, seed.PlayedAt); err != nil {
		t.Fatalf("insert player_match_stats row: %v", err)
	}
}

func fmtDemoName(prefix string, index int) string {
	return fmt.Sprintf("%s-%03d.mvd", strings.ToLower(prefix), index)
}
