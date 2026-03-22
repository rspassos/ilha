package storage

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rspassos/ilha/jobs/collector/internal/model"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Repository struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

type UpsertResult struct {
	Inserted int
	Updated  int
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
		now:  time.Now().UTC,
	}
}

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(strings.TrimSpace(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func (r *Repository) ApplyMigrations(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return errors.New("repository is not initialized")
	}

	statements := []string{
		"CREATE TABLE IF NOT EXISTS collector_schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL)",
	}
	for _, statement := range statements {
		if _, err := r.pool.Exec(ctx, statement); err != nil {
			return fmt.Errorf("apply bootstrap migration statement: %w", err)
		}
	}

	data, err := migrationFiles.ReadFile("migrations/001_create_collector_matches.sql")
	if err != nil {
		return fmt.Errorf("read embedded migration: %w", err)
	}

	var alreadyApplied bool
	if err := r.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM collector_schema_migrations WHERE version = $1)", "001_create_collector_matches").Scan(&alreadyApplied); err != nil {
		return fmt.Errorf("check migration state: %w", err)
	}
	if alreadyApplied {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, string(data)); err != nil {
		return fmt.Errorf("apply migration 001_create_collector_matches: %w", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO collector_schema_migrations(version, applied_at) VALUES ($1, $2)", "001_create_collector_matches", r.now()); err != nil {
		return fmt.Errorf("record migration 001_create_collector_matches: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration transaction: %w", err)
	}

	return nil
}

func (r *Repository) UpsertMatches(ctx context.Context, matches []model.MatchRecord) (UpsertResult, error) {
	if r == nil || r.pool == nil {
		return UpsertResult{}, errors.New("repository is not initialized")
	}
	if len(matches) == 0 {
		return UpsertResult{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return UpsertResult{}, fmt.Errorf("begin upsert transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	batch := &pgx.Batch{}
	for _, match := range matches {
		batch.Queue(upsertMatchSQL,
			match.ServerKey,
			match.ServerName,
			match.DemoName,
			match.MatchKey,
			match.Mode,
			match.MapName,
			nullIfEmpty(match.Participants),
			match.PlayedAt,
			nullInt(match.DurationSeconds),
			nullIfEmpty(match.Hostname),
			match.HasBots,
			match.ScorePayload,
			match.StatsPayload,
			match.MergedPayload,
			r.now(),
			r.now(),
		)
	}

	results := tx.SendBatch(ctx, batch)

	var summary UpsertResult
	for _, match := range matches {
		var inserted bool
		if err := results.QueryRow().Scan(&inserted); err != nil {
			return UpsertResult{}, fmt.Errorf("upsert match %q: %w", match.MatchKey, err)
		}
		if inserted {
			summary.Inserted++
		} else {
			summary.Updated++
		}
	}

	if err := results.Close(); err != nil {
		return UpsertResult{}, fmt.Errorf("close upsert batch results: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return UpsertResult{}, fmt.Errorf("commit upsert transaction: %w", err)
	}

	return summary, nil
}

func nullIfEmpty(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

const upsertMatchSQL = `
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
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13::jsonb, $14::jsonb, $15, $16
)
ON CONFLICT (server_key, demo_name) DO UPDATE SET
	server_name = EXCLUDED.server_name,
	match_key = EXCLUDED.match_key,
	mode = EXCLUDED.mode,
	map_name = EXCLUDED.map_name,
	participants = EXCLUDED.participants,
	played_at = EXCLUDED.played_at,
	duration_seconds = EXCLUDED.duration_seconds,
	hostname = EXCLUDED.hostname,
	has_bots = EXCLUDED.has_bots,
	score_payload = EXCLUDED.score_payload,
	stats_payload = EXCLUDED.stats_payload,
	merged_payload = EXCLUDED.merged_payload,
	updated_at = EXCLUDED.updated_at
RETURNING (xmax = 0) AS inserted
`
