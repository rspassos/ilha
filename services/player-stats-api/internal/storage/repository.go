package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rspassos/ilha/services/player-stats-api/internal/logging"
	"github.com/rspassos/ilha/services/player-stats-api/internal/metrics"
	"github.com/rspassos/ilha/services/player-stats-api/internal/model"
)

var (
	errRepositoryNotInitialized = errors.New("repository is not initialized")
	errInvalidLimit             = errors.New("query limit must be greater than zero")
	errInvalidOffset            = errors.New("query offset must be greater than or equal to zero")
	errInvalidMinimumMatches    = errors.New("query minimum_matches must be greater than zero")
)

var sortColumns = map[string]string{
	"efficiency":  "efficiency",
	"frags":       "frags",
	"lg_accuracy": "lg_accuracy",
	"rl_hits":     "rl_hits",
}

type RankingRepository interface {
	ListPlayerRanking(ctx context.Context, query model.RankingQuery) (model.RankingPage, error)
}

type Repository struct {
	pool    *Pool
	logger  *logging.Logger
	metrics *metrics.Collector
}

func NewRepository(pool *Pool, logger *logging.Logger, metricsCollector *metrics.Collector) *Repository {
	return &Repository{
		pool:    pool,
		logger:  logger,
		metrics: metricsCollector,
	}
}

func (r *Repository) ListPlayerRanking(ctx context.Context, query model.RankingQuery) (model.RankingPage, error) {
	if r == nil || r.pool == nil {
		return model.RankingPage{}, errRepositoryNotInitialized
	}

	query = normalizeQuery(query)
	if err := validateQuery(query); err != nil {
		return model.RankingPage{}, err
	}

	orderBy, err := buildOrderByClause(query.SortBy, query.SortDirection)
	if err != nil {
		return model.RankingPage{}, err
	}

	sql, args := buildListRankingSQL(query, orderBy)
	startedAt := time.Now()
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		r.recordQueryResult("error", time.Since(startedAt), err)
		return model.RankingPage{}, fmt.Errorf("list player ranking: %w", err)
	}
	defer rows.Close()

	result := make([]model.PlayerRankingRow, 0, query.Limit+1)
	for rows.Next() {
		var row model.PlayerRankingRow
		if err := rows.Scan(
			&row.PlayerID,
			&row.DisplayName,
			&row.Matches,
			&row.Efficiency,
			&row.Frags,
			&row.Kills,
			&row.Deaths,
			&row.LGAccuracy,
			&row.RLHits,
			&row.Rank,
		); err != nil {
			return model.RankingPage{}, fmt.Errorf("scan player ranking row: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		r.recordQueryResult("error", time.Since(startedAt), err)
		return model.RankingPage{}, fmt.Errorf("iterate player ranking rows: %w", err)
	}
	r.recordQueryResult("success", time.Since(startedAt), nil)

	hasNext := len(result) > query.Limit
	if hasNext {
		result = result[:query.Limit]
	}

	return model.NewRankingPage(query, result, hasNext), nil
}

func (r *Repository) recordQueryResult(status string, duration time.Duration, queryErr error) {
	if r.metrics != nil {
		r.metrics.RecordDBQuery("list_player_ranking", status)
	}

	if queryErr != nil && r.logger != nil {
		_ = r.logger.Error("player stats api database query failed", map[string]any{
			"query":       "list_player_ranking",
			"status":      status,
			"duration_ms": duration.Milliseconds(),
			"error":       queryErr.Error(),
		})
	}
}

func normalizeQuery(query model.RankingQuery) model.RankingQuery {
	query.Mode = strings.TrimSpace(query.Mode)
	query.Map = strings.TrimSpace(query.Map)
	query.Server = strings.TrimSpace(query.Server)
	query.SortBy = strings.ToLower(strings.TrimSpace(query.SortBy))
	query.SortDirection = strings.ToLower(strings.TrimSpace(query.SortDirection))

	if query.SortBy == "" {
		query.SortBy = model.DefaultSortBy
	}
	if query.SortDirection == "" {
		query.SortDirection = model.DefaultSortDirection
	}

	if !query.From.IsZero() {
		query.From = query.From.UTC()
	}
	if !query.To.IsZero() {
		query.To = query.To.UTC()
	}

	return query
}

func validateQuery(query model.RankingQuery) error {
	if query.Limit <= 0 {
		return errInvalidLimit
	}
	if query.Offset < 0 {
		return errInvalidOffset
	}
	if query.MinimumMatches <= 0 {
		return errInvalidMinimumMatches
	}

	return nil
}

func buildOrderByClause(sortBy string, sortDirection string) (string, error) {
	column, ok := sortColumns[sortBy]
	if !ok {
		return "", fmt.Errorf("unsupported sort_by %q", sortBy)
	}

	switch sortDirection {
	case "asc", "desc":
	default:
		return "", fmt.Errorf("unsupported sort_direction %q", sortDirection)
	}

	orderByParts := []string{
		fmt.Sprintf("%s %s", column, strings.ToUpper(sortDirection)),
	}
	if sortBy != "frags" {
		orderByParts = append(orderByParts, "frags DESC")
	}
	orderByParts = append(orderByParts, "matches DESC", "player_id ASC")

	return strings.Join(orderByParts, ", "), nil
}

func buildListRankingSQL(query model.RankingQuery, orderBy string) (string, []any) {
	args := make([]any, 0, 8)
	clauses := []string{"stats.excluded_from_analytics = FALSE"}

	if query.Mode != "" {
		args = append(args, query.Mode)
		clauses = append(clauses, fmt.Sprintf("stats.normalized_mode = $%d", len(args)))
	}
	if query.Map != "" {
		args = append(args, query.Map)
		clauses = append(clauses, fmt.Sprintf("stats.map_name = $%d", len(args)))
	}
	if query.Server != "" {
		args = append(args, query.Server)
		clauses = append(clauses, fmt.Sprintf("stats.server_key = $%d", len(args)))
	}
	if !query.From.IsZero() {
		args = append(args, query.From)
		clauses = append(clauses, fmt.Sprintf("stats.played_at >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		clauses = append(clauses, fmt.Sprintf("stats.played_at <= $%d", len(args)))
	}

	args = append(args, query.MinimumMatches, query.Limit+1, query.Offset)
	minimumMatchesArg := len(args) - 2
	limitArg := len(args) - 1
	offsetArg := len(args)

	sql := fmt.Sprintf(`
WITH aggregated AS (
	SELECT
		stats.player_id::text AS player_id,
		canonical.display_name,
		COUNT(*)::integer AS matches,
		ROUND(AVG(stats.efficiency), 2)::double precision AS efficiency,
		COALESCE(SUM(stats.frags), 0)::integer AS frags,
		COALESCE(SUM(stats.kills), 0)::integer AS kills,
		COALESCE(SUM(stats.deaths), 0)::integer AS deaths,
		ROUND(AVG(stats.lg_accuracy), 2)::double precision AS lg_accuracy,
		COALESCE(SUM(stats.rl_hits), 0)::integer AS rl_hits
	FROM player_match_stats stats
	JOIN player_canonical canonical ON canonical.id = stats.player_id
	WHERE %s
	GROUP BY stats.player_id, canonical.display_name
	HAVING COUNT(*) >= $%d
),
ranked AS (
	SELECT
		player_id,
		display_name,
		matches,
		efficiency,
		frags,
		kills,
		deaths,
		lg_accuracy,
		rl_hits,
		ROW_NUMBER() OVER (ORDER BY %s) AS rank
	FROM aggregated
)
SELECT
	player_id,
	display_name,
	matches,
	efficiency,
	frags,
	kills,
	deaths,
	lg_accuracy,
	rl_hits,
	rank
FROM ranked
ORDER BY %s
LIMIT $%d OFFSET $%d
`, strings.Join(clauses, " AND "), minimumMatchesArg, orderBy, orderBy, limitArg, offsetArg)

	return sql, args
}
