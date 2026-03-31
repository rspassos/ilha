package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rspassos/ilha/services/player-stats-api/internal/logging"
	"github.com/rspassos/ilha/services/player-stats-api/internal/metrics"
	"github.com/rspassos/ilha/services/player-stats-api/internal/model"
	"github.com/rspassos/ilha/services/player-stats-api/internal/service"
)

var (
	allowedModes = map[string]struct{}{
		"1on1": {},
		"2on2": {},
		"3on3": {},
		"4on4": {},
		"dmm4": {},
	}
	allowedSortBy = map[string]struct{}{
		"efficiency":  {},
		"frags":       {},
		"lg_accuracy": {},
		"rl_hits":     {},
	}
	allowedSortDirections = map[string]struct{}{
		"asc":  {},
		"desc": {},
	}
)

type RankingQueryParser struct {
	defaultLimit   int
	maxLimit       int
	minimumMatches int
	defaultSortBy  string
	defaultSortDir string
}

type RankingHandler struct {
	service       service.RankingService
	parser        RankingQueryParser
	problemWriter ProblemWriter
	logger        *logging.Logger
	metrics       *metrics.Collector
}

func NewRankingQueryParser(defaultLimit int, maxLimit int, minimumMatches int) RankingQueryParser {
	return RankingQueryParser{
		defaultLimit:   defaultLimit,
		maxLimit:       maxLimit,
		minimumMatches: minimumMatches,
		defaultSortBy:  model.DefaultSortBy,
		defaultSortDir: model.DefaultSortDirection,
	}
}

func NewRankingHandler(
	service service.RankingService,
	parser RankingQueryParser,
	problemWriter ProblemWriter,
	logger *logging.Logger,
	metricsCollector *metrics.Collector,
) *RankingHandler {
	return &RankingHandler{
		service:       service,
		parser:        parser,
		problemWriter: problemWriter,
		logger:        logger,
		metrics:       metricsCollector,
	}
}

func (h *RankingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		h.problemWriter.WriteProblem(w, http.StatusMethodNotAllowed, Problem{
			Type:   "https://ilha.dev/problems/method-not-allowed",
			Title:  "Method not allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: "only GET is supported for this endpoint",
		})
		return
	}

	query, err := h.parser.Parse(r)
	if err != nil {
		var validationErr *QueryValidationError
		if errors.As(err, &validationErr) {
			h.addRequestFields(r.Context(), map[string]any{
				"invalid_params": validationErr.InvalidParams,
			})
			if h.metrics != nil {
				for _, invalid := range validationErr.InvalidParams {
					h.metrics.RecordInvalidRequest(invalid.Name)
				}
			}
			if h.logger != nil {
				_ = h.logger.Warn("player stats api request validation failed", map[string]any{
					"endpoint":       r.URL.Path,
					"invalid_params": validationErr.InvalidParams,
				})
			}
			h.problemWriter.WriteProblem(w, http.StatusBadRequest, Problem{
				Type:          "https://ilha.dev/problems/invalid-query",
				Title:         "Invalid query parameters",
				Status:        http.StatusBadRequest,
				Detail:        "one or more query parameters are invalid",
				Instance:      r.URL.Path,
				InvalidParams: validationErr.InvalidParams,
			})
			return
		}

		h.writeInternalProblem(w, r.URL.Path)
		return
	}

	page, err := h.service.ListPlayerRanking(r.Context(), query)
	if err != nil {
		if h.logger != nil {
			_ = h.logger.Error("player stats api request failed", map[string]any{
				"endpoint": r.URL.Path,
				"error":    err.Error(),
			})
		}
		h.writeInternalProblem(w, r.URL.Path)
		return
	}

	if h.metrics != nil {
		h.metrics.RecordRankingRowsReturned(len(page.Data))
	}
	h.addRequestFields(r.Context(), map[string]any{
		"filters": map[string]any{
			"mode":   page.Filters.Mode,
			"map":    page.Filters.Map,
			"server": page.Filters.Server,
			"from":   page.Filters.From,
			"to":     page.Filters.To,
		},
		"sort_by":        page.Meta.SortBy,
		"sort_direction": page.Meta.SortDirection,
		"limit":          page.Meta.Limit,
		"offset":         page.Meta.Offset,
		"returned_rows":  len(page.Data),
		"has_next":       page.Meta.HasNext,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(page)
}

func (h *RankingHandler) writeInternalProblem(w http.ResponseWriter, instance string) {
	h.problemWriter.WriteProblem(w, http.StatusInternalServerError, Problem{
		Type:     "https://ilha.dev/problems/internal-server-error",
		Title:    "Internal server error",
		Status:   http.StatusInternalServerError,
		Detail:   "the server could not process the request",
		Instance: instance,
	})
}

func (h *RankingHandler) addRequestFields(ctx context.Context, fields map[string]any) {
	requestFields := logging.RequestFieldsFromContext(ctx)
	if requestFields == nil {
		return
	}

	requestFields.Add(fields)
}

func (p RankingQueryParser) Parse(r *http.Request) (model.RankingQuery, error) {
	values := r.URL.Query()
	invalidParams := make([]InvalidParam, 0)

	query := model.RankingQuery{
		SortBy:         p.defaultSortBy,
		SortDirection:  p.defaultSortDir,
		Limit:          p.defaultLimit,
		Offset:         0,
		MinimumMatches: p.minimumMatches,
	}

	if value := strings.TrimSpace(values.Get("mode")); value != "" {
		normalized := strings.ToLower(value)
		if _, ok := allowedModes[normalized]; !ok {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "mode",
				Reason: "must be one of 1on1, 2on2, 3on3, 4on4, dmm4",
				Value:  value,
			})
		} else {
			query.Mode = normalized
		}
	}

	if value := strings.TrimSpace(values.Get("map")); value != "" {
		query.Map = value
	}

	if value := strings.TrimSpace(values.Get("server")); value != "" {
		query.Server = value
	}

	if value := strings.TrimSpace(values.Get("from")); value != "" {
		parsed, err := parseTimeQueryValue(value, false)
		if err != nil {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "from",
				Reason: "must be RFC3339 or YYYY-MM-DD",
				Value:  value,
			})
		} else {
			query.From = parsed
		}
	}

	if value := strings.TrimSpace(values.Get("to")); value != "" {
		parsed, err := parseTimeQueryValue(value, true)
		if err != nil {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "to",
				Reason: "must be RFC3339 or YYYY-MM-DD",
				Value:  value,
			})
		} else {
			query.To = parsed
		}
	}

	if !query.From.IsZero() && !query.To.IsZero() && query.From.After(query.To) {
		invalidParams = append(invalidParams, InvalidParam{
			Name:   "from",
			Reason: "must be before or equal to to",
			Value:  query.From.Format(time.RFC3339),
		})
	}

	if value := strings.TrimSpace(values.Get("sort_by")); value != "" {
		normalized := strings.ToLower(value)
		if _, ok := allowedSortBy[normalized]; !ok {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "sort_by",
				Reason: "must be one of efficiency, frags, lg_accuracy, rl_hits",
				Value:  value,
			})
		} else {
			query.SortBy = normalized
		}
	}

	if value := strings.TrimSpace(values.Get("sort_direction")); value != "" {
		normalized := strings.ToLower(value)
		if _, ok := allowedSortDirections[normalized]; !ok {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "sort_direction",
				Reason: "must be one of asc, desc",
				Value:  value,
			})
		} else {
			query.SortDirection = normalized
		}
	}

	if value := strings.TrimSpace(values.Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "limit",
				Reason: "must be an integer",
				Value:  value,
			})
		} else if parsed <= 0 || parsed > p.maxLimit {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "limit",
				Reason: fmt.Sprintf("must be between 1 and %d", p.maxLimit),
				Value:  value,
			})
		} else {
			query.Limit = parsed
		}
	}

	if value := strings.TrimSpace(values.Get("offset")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "offset",
				Reason: "must be an integer",
				Value:  value,
			})
		} else if parsed < 0 {
			invalidParams = append(invalidParams, InvalidParam{
				Name:   "offset",
				Reason: "must be greater than or equal to 0",
				Value:  value,
			})
		} else {
			query.Offset = parsed
		}
	}

	if len(invalidParams) > 0 {
		return model.RankingQuery{}, &QueryValidationError{InvalidParams: invalidParams}
	}

	return query, nil
}

type QueryValidationError struct {
	InvalidParams []InvalidParam
}

func (e *QueryValidationError) Error() string {
	return "invalid query parameters"
}

func parseTimeQueryValue(value string, endOfDay bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return parsed.Add(24*time.Hour - time.Nanosecond).UTC(), nil
	}

	return parsed.UTC(), nil
}

func NewHandler(defaultLimit int, maxLimit int, minimumMatches int, rankingService service.RankingService, logger *logging.Logger, metricsCollector *metrics.Collector) http.Handler {
	problemWriter := NewProblemWriter()
	parser := NewRankingQueryParser(defaultLimit, maxLimit, minimumMatches)
	return NewRankingHandler(rankingService, parser, problemWriter, logger, metricsCollector)
}

type noopRankingService struct{}

func NewNoopRankingService() service.RankingService {
	return noopRankingService{}
}

func (noopRankingService) ListPlayerRanking(_ context.Context, query model.RankingQuery) (model.RankingPage, error) {
	return model.NewRankingPage(query, []model.PlayerRankingRow{}, false), nil
}
