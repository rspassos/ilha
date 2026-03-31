package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rspassos/ilha/services/player-stats-api/internal/logging"
	"github.com/rspassos/ilha/services/player-stats-api/internal/metrics"
	"github.com/rspassos/ilha/services/player-stats-api/internal/model"
	"github.com/rspassos/ilha/services/player-stats-api/internal/service"
)

func TestRankingQueryParserParseAppliesDefaults(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players", nil)
	parser := NewRankingQueryParser(50, 100, 10)

	query, err := parser.Parse(req)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if query.SortBy != model.DefaultSortBy {
		t.Fatalf("SortBy = %q, want %q", query.SortBy, model.DefaultSortBy)
	}
	if query.SortDirection != model.DefaultSortDirection {
		t.Fatalf("SortDirection = %q, want %q", query.SortDirection, model.DefaultSortDirection)
	}
	if query.Limit != 50 {
		t.Fatalf("Limit = %d, want 50", query.Limit)
	}
	if query.Offset != 0 {
		t.Fatalf("Offset = %d, want 0", query.Offset)
	}
	if query.MinimumMatches != 10 {
		t.Fatalf("MinimumMatches = %d, want 10", query.MinimumMatches)
	}
}

func TestRankingQueryParserParseAcceptsSupportedParams(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players?mode=2ON2&map=%20aerowalk%20&server=%20alpha%20&from=2026-03-01&to=2026-03-31&sort_by=LG_ACCURACY&sort_direction=ASC&limit=20&offset=40", nil)
	parser := NewRankingQueryParser(50, 100, 10)

	query, err := parser.Parse(req)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if query.Mode != "2on2" {
		t.Fatalf("Mode = %q, want 2on2", query.Mode)
	}
	if query.Map != "aerowalk" {
		t.Fatalf("Map = %q, want aerowalk", query.Map)
	}
	if query.Server != "alpha" {
		t.Fatalf("Server = %q, want alpha", query.Server)
	}
	if query.SortBy != "lg_accuracy" {
		t.Fatalf("SortBy = %q, want lg_accuracy", query.SortBy)
	}
	if query.SortDirection != "asc" {
		t.Fatalf("SortDirection = %q, want asc", query.SortDirection)
	}
	if query.Limit != 20 {
		t.Fatalf("Limit = %d, want 20", query.Limit)
	}
	if query.Offset != 40 {
		t.Fatalf("Offset = %d, want 40", query.Offset)
	}

	wantFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if !query.From.Equal(wantFrom) {
		t.Fatalf("From = %s, want %s", query.From.Format(time.RFC3339), wantFrom.Format(time.RFC3339))
	}

	wantTo := time.Date(2026, 3, 31, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
	if !query.To.Equal(wantTo) {
		t.Fatalf("To = %s, want %s", query.To.Format(time.RFC3339Nano), wantTo.Format(time.RFC3339Nano))
	}
}

func TestRankingQueryParserParseRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players?mode=ctf&from=bad-date&sort_by=kills&sort_direction=down&limit=0&offset=-1", nil)
	parser := NewRankingQueryParser(50, 100, 10)

	_, err := parser.Parse(req)
	if err == nil {
		t.Fatal("Parse() error = nil, want non-nil")
	}

	var validationErr *QueryValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Parse() error = %T, want *QueryValidationError", err)
	}
	if len(validationErr.InvalidParams) != 6 {
		t.Fatalf("InvalidParams len = %d, want 6", len(validationErr.InvalidParams))
	}
}

func TestRankingQueryParserParseRejectsLimitAboveMax(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players?limit=101", nil)
	parser := NewRankingQueryParser(50, 100, 10)

	_, err := parser.Parse(req)
	if err == nil {
		t.Fatal("Parse() error = nil, want non-nil")
	}
}

func TestRankingQueryParserParseRejectsFromAfterTo(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players?from=2026-04-01&to=2026-03-31", nil)
	parser := NewRankingQueryParser(50, 100, 10)

	_, err := parser.Parse(req)
	if err == nil {
		t.Fatal("Parse() error = nil, want non-nil")
	}
}

func TestRankingHandlerServeHTTPReturnsRankingPage(t *testing.T) {
	t.Parallel()

	var gotQuery model.RankingQuery
	handler := NewHandler(
		50,
		100,
		10,
		service.RankingServiceFunc(func(_ context.Context, query model.RankingQuery) (model.RankingPage, error) {
			gotQuery = query
			rows := []model.PlayerRankingRow{
				{
					PlayerID:    "player-1",
					DisplayName: "Player One",
					Matches:     12,
					Efficiency:  55.5,
					Frags:       210,
					Kills:       190,
					Deaths:      152,
					LGAccuracy:  38.4,
					RLHits:      88,
					Rank:        1,
				},
			}
			return model.NewRankingPage(query, rows, true), nil
		}),
		logging.New(io.Discard, "player-stats-api"),
		metrics.New(),
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players?mode=1on1&sort_by=frags&limit=1", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if got := recorder.Code; got != http.StatusOK {
		t.Fatalf("status = %d, want %d", got, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	if gotQuery.Mode != "1on1" {
		t.Fatalf("service query mode = %q, want 1on1", gotQuery.Mode)
	}
	if gotQuery.SortBy != "frags" {
		t.Fatalf("service query sort_by = %q, want frags", gotQuery.SortBy)
	}

	var page model.RankingPage
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(page.Data))
	}
	if page.Meta.MinimumMatches != 10 {
		t.Fatalf("minimum_matches = %d, want 10", page.Meta.MinimumMatches)
	}
	if !page.Meta.HasNext {
		t.Fatal("has_next = false, want true")
	}
	if page.Filters.Mode != "1on1" {
		t.Fatalf("filters.mode = %q, want 1on1", page.Filters.Mode)
	}
}

func TestRankingHandlerServeHTTPReturnsValidationProblem(t *testing.T) {
	t.Parallel()

	handler := NewHandler(50, 100, 10, NewNoopRankingService(), logging.New(io.Discard, "player-stats-api"), metrics.New())
	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players?sort_by=kills", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if got := recorder.Code; got != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", got, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", got)
	}

	var problem Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal problem: %v", err)
	}
	if problem.Title != "Invalid query parameters" {
		t.Fatalf("title = %q, want Invalid query parameters", problem.Title)
	}
	if len(problem.InvalidParams) != 1 {
		t.Fatalf("invalid_params len = %d, want 1", len(problem.InvalidParams))
	}
	if problem.InvalidParams[0].Name != "sort_by" {
		t.Fatalf("invalid_params[0].name = %q, want sort_by", problem.InvalidParams[0].Name)
	}
}

func TestRankingHandlerServeHTTPLogsValidationFailure(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	handler := NewHandler(
		50,
		100,
		10,
		NewNoopRankingService(),
		logging.New(&logBuffer, "player-stats-api"),
		metrics.New(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players?sort_by=kills", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if got := recorder.Code; got != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", got, http.StatusBadRequest)
	}
	if got := logBuffer.String(); !strings.Contains(got, `"message":"player stats api request validation failed"`) {
		t.Fatalf("log output missing validation failure message: %s", got)
	}
}

func TestRankingHandlerServeHTTPReturnsInternalProblem(t *testing.T) {
	t.Parallel()

	handler := NewHandler(
		50,
		100,
		10,
		service.RankingServiceFunc(func(context.Context, model.RankingQuery) (model.RankingPage, error) {
			return model.RankingPage{}, errors.New("boom")
		}),
		logging.New(io.Discard, "player-stats-api"),
		metrics.New(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if got := recorder.Code; got != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", got, http.StatusInternalServerError)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", got)
	}
}

func TestRankingHandlerServeHTTPLogsInternalFailure(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	handler := NewHandler(
		50,
		100,
		10,
		service.RankingServiceFunc(func(context.Context, model.RankingQuery) (model.RankingPage, error) {
			return model.RankingPage{}, errors.New("boom")
		}),
		logging.New(&logBuffer, "player-stats-api"),
		metrics.New(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/rankings/players", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if got := recorder.Code; got != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", got, http.StatusInternalServerError)
	}
	if got := logBuffer.String(); !strings.Contains(got, `"message":"player stats api request failed"`) {
		t.Fatalf("log output missing internal failure message: %s", got)
	}
}

func TestRankingHandlerServeHTTPMethodNotAllowedIncludesAllowHeader(t *testing.T) {
	t.Parallel()

	handler := NewHandler(50, 100, 10, NewNoopRankingService(), logging.New(io.Discard, "player-stats-api"), metrics.New())
	req := httptest.NewRequest(http.MethodPost, "/v1/rankings/players", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if got := recorder.Code; got != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", got, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("allow = %q, want %q", got, http.MethodGet)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", got)
	}
}
