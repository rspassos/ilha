package storage

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/rspassos/ilha/jobs/player-stats/internal/identity"
	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Repository struct {
	pool *Pool
	now  func() time.Time
}

func NewRepository(pool *Pool) *Repository {
	return &Repository{
		pool: pool,
		now:  time.Now().UTC,
	}
}

func (r *Repository) ApplyMigrations(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return errors.New("repository is not initialized")
	}

	if _, err := r.pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto"); err != nil {
		return fmt.Errorf("create pgcrypto extension: %w", err)
	}

	if _, err := r.pool.Exec(ctx, "CREATE TABLE IF NOT EXISTS player_stats_schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL)"); err != nil {
		return fmt.Errorf("apply bootstrap migration statement: %w", err)
	}

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		version := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

		var alreadyApplied bool
		if err := r.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM player_stats_schema_migrations WHERE version = $1)", version).Scan(&alreadyApplied); err != nil {
			return fmt.Errorf("check migration state %s: %w", version, err)
		}
		if alreadyApplied {
			continue
		}

		data, err := migrationFiles.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration transaction %s: %w", version, err)
		}

		if err := applyMigration(ctx, tx, string(data), version, r.now()); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration transaction %s: %w", version, err)
		}
	}

	return nil
}

func applyMigration(ctx context.Context, tx pgx.Tx, sql string, version string, appliedAt time.Time) error {
	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("apply migration %s: %w", version, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO player_stats_schema_migrations(version, applied_at) VALUES ($1, $2)", version, appliedAt); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	return nil
}

func (r *Repository) LoadCheckpoint(ctx context.Context, jobName string) (model.Checkpoint, error) {
	if r == nil || r.pool == nil {
		return model.Checkpoint{}, errors.New("repository is not initialized")
	}

	var checkpoint model.Checkpoint
	err := r.pool.QueryRow(ctx, `
		SELECT job_name, last_collector_match_id, updated_at
		FROM player_stats_checkpoints
		WHERE job_name = $1
	`, strings.TrimSpace(jobName)).Scan(&checkpoint.JobName, &checkpoint.LastCollectorMatchID, &checkpoint.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Checkpoint{
				JobName:              strings.TrimSpace(jobName),
				LastCollectorMatchID: 0,
			}, nil
		}
		return model.Checkpoint{}, fmt.Errorf("load checkpoint %q: %w", jobName, err)
	}

	return checkpoint, nil
}

func (r *Repository) SaveCheckpoint(ctx context.Context, checkpoint model.Checkpoint) error {
	if r == nil || r.pool == nil {
		return errors.New("repository is not initialized")
	}

	if strings.TrimSpace(checkpoint.JobName) == "" {
		return errors.New("checkpoint job_name must not be empty")
	}

	updatedAt := checkpoint.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = r.now()
	}

	if _, err := r.pool.Exec(ctx, checkpointUpsertSQL, checkpoint.JobName, checkpoint.LastCollectorMatchID, updatedAt); err != nil {
		return fmt.Errorf("save checkpoint %q: %w", checkpoint.JobName, err)
	}

	return nil
}

func (r *Repository) UpsertBatch(ctx context.Context, batch model.ConsolidationBatch) (model.BatchResult, error) {
	if r == nil || r.pool == nil {
		return model.BatchResult{}, errors.New("repository is not initialized")
	}
	if strings.TrimSpace(batch.Checkpoint.JobName) == "" {
		return model.BatchResult{}, errors.New("batch checkpoint job_name must not be empty")
	}
	if len(batch.Rows) == 0 {
		if err := r.SaveCheckpoint(ctx, batch.Checkpoint); err != nil {
			return model.BatchResult{}, err
		}
		return model.BatchResult{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.BatchResult{}, fmt.Errorf("begin analytics upsert transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var result model.BatchResult
	for _, row := range batch.Rows {
		playerIdentity, err := resolvePlayerIdentity(ctx, tx, row, r.now())
		if err != nil {
			return model.BatchResult{}, err
		}
		applyIdentityResult(&result, playerIdentity)

		inserted, err := upsertPlayerMatchStats(ctx, tx, row, playerIdentity.PlayerID)
		if err != nil {
			return model.BatchResult{}, err
		}
		if inserted {
			result.StatsInserted++
		} else {
			result.StatsUpdated++
		}
	}

	checkpointUpdatedAt := batch.Checkpoint.UpdatedAt
	if checkpointUpdatedAt.IsZero() {
		checkpointUpdatedAt = r.now()
	}
	if _, err := tx.Exec(ctx, checkpointUpsertSQL, batch.Checkpoint.JobName, batch.Checkpoint.LastCollectorMatchID, checkpointUpdatedAt); err != nil {
		return model.BatchResult{}, fmt.Errorf("save batch checkpoint %q: %w", batch.Checkpoint.JobName, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return model.BatchResult{}, fmt.Errorf("commit analytics upsert transaction: %w", err)
	}

	return result, nil
}

func (r *Repository) ResolvePlayer(ctx context.Context, input model.ResolvePlayerInput) (model.PlayerIdentity, error) {
	if r == nil || r.pool == nil {
		return model.PlayerIdentity{}, errors.New("repository is not initialized")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.PlayerIdentity{}, fmt.Errorf("begin identity transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	resolver := identity.NewResolver(txIdentityStore{tx: tx})
	playerIdentity, err := resolver.ResolvePlayer(ctx, input)
	if err != nil {
		return model.PlayerIdentity{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.PlayerIdentity{}, fmt.Errorf("commit identity transaction: %w", err)
	}

	return playerIdentity, nil
}

func resolvePlayerIdentity(ctx context.Context, tx pgx.Tx, row model.PlayerMatchRow, now time.Time) (model.PlayerIdentity, error) {
	resolver := identity.NewResolver(txIdentityStore{
		tx:               tx,
		collectorMatchID: row.CollectorMatchID,
	})

	playerIdentity, err := resolver.ResolvePlayer(ctx, model.ResolvePlayerInput{
		ObservedName:  row.ObservedName,
		ObservedLogin: row.ObservedLogin,
		ObservedAt:    coalesceTime(row.PlayedAt, now),
	})
	if err != nil {
		return model.PlayerIdentity{}, fmt.Errorf("resolve identity for match %d: %w", row.CollectorMatchID, err)
	}

	return playerIdentity, nil
}

func applyIdentityResult(result *model.BatchResult, playerIdentity model.PlayerIdentity) {
	if result == nil {
		return
	}

	switch playerIdentity.Resolution {
	case identity.ResolutionCreated:
		result.CanonicalInserted++
	case identity.ResolutionByLogin, identity.ResolutionByAlias:
		result.CanonicalReused++
	}

	switch playerIdentity.AliasAction {
	case identity.AliasActionInserted:
		result.AliasesInserted++
	case identity.AliasActionUpdated:
		result.AliasesUpdated++
	}
}

func upsertPlayerMatchStats(ctx context.Context, tx pgx.Tx, row model.PlayerMatchRow, playerID string) (bool, error) {
	var inserted bool
	err := tx.QueryRow(ctx, upsertPlayerMatchStatsSQL,
		row.CollectorMatchID,
		playerID,
		row.ServerKey,
		row.DemoName,
		row.ObservedName,
		nullIfEmpty(row.ObservedLogin),
		nullIfEmpty(row.Team),
		row.MapName,
		nullIfEmpty(row.RawMode),
		row.NormalizedMode,
		row.PlayedAt,
		row.HasBots,
		row.ExcludedFromAnalytics,
		row.Frags,
		row.Deaths,
		row.Kills,
		row.TeamKills,
		row.Suicides,
		row.DamageTaken,
		row.DamageGiven,
		row.SpreeMax,
		row.SpreeQuad,
		row.RLHits,
		row.RLKills,
		row.LGAttacks,
		row.LGHits,
		row.GA,
		row.RA,
		row.YA,
		row.Health100,
		row.Ping,
		row.Efficiency,
		row.LGAccuracy,
		row.StatsSnapshot,
		coalesceTime(row.ConsolidatedAt, row.PlayedAt),
	).Scan(&inserted)
	if err != nil {
		return false, fmt.Errorf("upsert player match stats for match %d player %s: %w", row.CollectorMatchID, playerID, err)
	}

	return inserted, nil
}

func nullIfEmpty(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func coalesceTime(primary time.Time, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary
	}
	return fallback
}

const checkpointUpsertSQL = `
INSERT INTO player_stats_checkpoints (job_name, last_collector_match_id, updated_at)
VALUES ($1, $2, $3)
ON CONFLICT (job_name) DO UPDATE
SET last_collector_match_id = EXCLUDED.last_collector_match_id,
    updated_at = EXCLUDED.updated_at
`

const aliasUpsertSQL = `
WITH updated AS (
	UPDATE player_aliases
	SET
		player_id = $1::uuid,
		first_seen_at = LEAST(player_aliases.first_seen_at, $4),
		last_seen_at = GREATEST(player_aliases.last_seen_at, $5)
	WHERE alias_name = $2
	  AND COALESCE(login, '') = COALESCE($3, '')
	RETURNING false AS inserted
),
inserted AS (
	INSERT INTO player_aliases (player_id, alias_name, login, first_seen_at, last_seen_at)
	SELECT $1::uuid, $2, $3, $4, $5
	WHERE NOT EXISTS (SELECT 1 FROM updated)
	RETURNING true AS inserted
)
SELECT inserted FROM inserted
UNION ALL
SELECT inserted FROM updated
LIMIT 1
`

const upsertPlayerMatchStatsSQL = `
WITH updated AS (
	UPDATE player_match_stats
	SET
		server_key = $3,
		demo_name = $4,
		observed_login = $6,
		team = $7,
		map_name = $8,
		raw_mode = $9,
		normalized_mode = $10,
		played_at = $11,
		has_bots = $12,
		excluded_from_analytics = $13,
		frags = $14,
		deaths = $15,
		kills = $16,
		team_kills = $17,
		suicides = $18,
		damage_taken = $19,
		damage_given = $20,
		spree_max = $21,
		spree_quad = $22,
		rl_hits = $23,
		rl_kills = $24,
		lg_attacks = $25,
		lg_hits = $26,
		ga = $27,
		ra = $28,
		ya = $29,
		health_100 = $30,
		ping = $31,
		efficiency = $32,
		lg_accuracy = $33,
		stats_snapshot = $34::jsonb,
		consolidated_at = $35
	WHERE collector_match_id = $1
	  AND player_id = $2::uuid
	  AND observed_name = $5
	RETURNING false AS inserted
),
inserted AS (
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
	SELECT
		$1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
		$19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34::jsonb, $35
	WHERE NOT EXISTS (SELECT 1 FROM updated)
	RETURNING true AS inserted
)
SELECT inserted FROM inserted
UNION ALL
SELECT inserted FROM updated
LIMIT 1
`
