package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
)

const (
	SkipReasonMissingStatsPayload = "missing_stats_payload"
	SkipReasonInvalidStatsPayload = "invalid_stats_payload"
)

type MatchSource interface {
	ListMatchesForConsolidation(ctx context.Context, cursor model.Cursor, limit int) ([]model.SourceMatch, model.Cursor, error)
}

type PostgresSource struct {
	pool rowQuerier
}

type rowQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func NewPostgresSource(pool *pgxpool.Pool) *PostgresSource {
	return &PostgresSource{pool: pool}
}

func (s *PostgresSource) ListMatchesForConsolidation(ctx context.Context, cursor model.Cursor, limit int) ([]model.SourceMatch, model.Cursor, error) {
	if s == nil || s.pool == nil {
		return nil, model.Cursor{}, errors.New("source is not initialized")
	}
	if limit <= 0 {
		return nil, model.Cursor{}, errors.New("limit must be greater than zero")
	}

	rows, err := s.pool.Query(ctx, listMatchesForConsolidationSQL, cursor.LastCollectorMatchID, limit)
	if err != nil {
		return nil, model.Cursor{}, fmt.Errorf("list matches for consolidation after collector_match_id %d: %w", cursor.LastCollectorMatchID, err)
	}
	defer rows.Close()

	matches := make([]model.SourceMatch, 0, limit)
	nextCursor := cursor
	for rows.Next() {
		match, err := scanSourceMatch(rows)
		if err != nil {
			return nil, model.Cursor{}, err
		}
		matches = append(matches, match)
		nextCursor.LastCollectorMatchID = match.CollectorMatchID
	}
	if err := rows.Err(); err != nil {
		return nil, model.Cursor{}, fmt.Errorf("iterate matches for consolidation: %w", err)
	}

	return matches, nextCursor, nil
}

func scanSourceMatch(rows pgx.Rows) (model.SourceMatch, error) {
	var match model.SourceMatch
	if err := rows.Scan(
		&match.CollectorMatchID,
		&match.ServerKey,
		&match.DemoName,
		&match.MapName,
		&match.RawMode,
		&match.PlayedAt,
		&match.HasBots,
		&match.StatsPayload,
	); err != nil {
		return model.SourceMatch{}, fmt.Errorf("scan source match row: %w", err)
	}

	match.SkipReason = classifySkipReason(match.StatsPayload, &match.Stats)
	return match, nil
}

func classifySkipReason(payload json.RawMessage, stats *model.SourceStatsMatch) string {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" || trimmed == "null" {
		return SkipReasonMissingStatsPayload
	}

	if err := json.Unmarshal(payload, stats); err != nil {
		return SkipReasonInvalidStatsPayload
	}
	if strings.TrimSpace(stats.Demo) == "" || len(stats.Players) == 0 {
		return SkipReasonInvalidStatsPayload
	}

	return ""
}

const listMatchesForConsolidationSQL = `
WITH checkpoint AS (
	SELECT played_at, id
	FROM collector_matches
	WHERE id = $1
)
SELECT
	id,
	server_key,
	demo_name,
	map_name,
	mode,
	played_at,
	has_bots,
	stats_payload
FROM collector_matches
WHERE
	CASE
		WHEN $1 <= 0 THEN true
		WHEN EXISTS (SELECT 1 FROM checkpoint) THEN (played_at, id) > (
			SELECT played_at, id
			FROM checkpoint
		)
		ELSE id > $1
	END
ORDER BY played_at ASC, id ASC
LIMIT $2
`
