package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rspassos/ilha/services/player-stats-api/internal/model"
)

func TestServiceListPlayerRankingAppliesDefaults(t *testing.T) {
	t.Parallel()

	repository := &stubRankingRepository{
		page: model.RankingPage{
			Data: []model.PlayerRankingRow{
				{PlayerID: "player-1", DisplayName: "Player One", Rank: 1},
			},
			Meta: model.PaginationMeta{HasNext: true},
		},
	}
	svc := newTestService(t, repository)

	page, err := svc.ListPlayerRanking(context.Background(), model.RankingQuery{})
	if err != nil {
		t.Fatalf("ListPlayerRanking() error = %v", err)
	}

	if repository.query.SortBy != model.DefaultSortBy {
		t.Fatalf("repository query sort_by = %q, want %q", repository.query.SortBy, model.DefaultSortBy)
	}
	if repository.query.SortDirection != model.DefaultSortDirection {
		t.Fatalf("repository query sort_direction = %q, want %q", repository.query.SortDirection, model.DefaultSortDirection)
	}
	if repository.query.Limit != 50 {
		t.Fatalf("repository query limit = %d, want 50", repository.query.Limit)
	}
	if repository.query.Offset != 0 {
		t.Fatalf("repository query offset = %d, want 0", repository.query.Offset)
	}
	if repository.query.MinimumMatches != 10 {
		t.Fatalf("repository query minimum_matches = %d, want 10", repository.query.MinimumMatches)
	}
	if page.Meta.SortBy != model.DefaultSortBy {
		t.Fatalf("meta sort_by = %q, want %q", page.Meta.SortBy, model.DefaultSortBy)
	}
	if page.Meta.SortDirection != model.DefaultSortDirection {
		t.Fatalf("meta sort_direction = %q, want %q", page.Meta.SortDirection, model.DefaultSortDirection)
	}
	if page.Meta.Limit != 50 {
		t.Fatalf("meta limit = %d, want 50", page.Meta.Limit)
	}
	if !page.Meta.HasNext {
		t.Fatal("meta has_next = false, want true")
	}
}

func TestServiceListPlayerRankingNormalizesFilters(t *testing.T) {
	t.Parallel()

	repository := &stubRankingRepository{}
	svc := newTestService(t, repository)

	from := time.Date(2026, 3, 1, 12, 0, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	to := time.Date(2026, 3, 31, 23, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))

	page, err := svc.ListPlayerRanking(context.Background(), model.RankingQuery{
		Mode:           " 2ON2 ",
		Map:            " aerowalk ",
		Server:         " alpha ",
		From:           from,
		To:             to,
		SortBy:         " LG_ACCURACY ",
		SortDirection:  " ASC ",
		Limit:          20,
		Offset:         40,
		MinimumMatches: 12,
	})
	if err != nil {
		t.Fatalf("ListPlayerRanking() error = %v", err)
	}

	if repository.query.Mode != "2on2" {
		t.Fatalf("repository query mode = %q, want 2on2", repository.query.Mode)
	}
	if repository.query.Map != "aerowalk" {
		t.Fatalf("repository query map = %q, want aerowalk", repository.query.Map)
	}
	if repository.query.Server != "alpha" {
		t.Fatalf("repository query server = %q, want alpha", repository.query.Server)
	}
	if repository.query.SortBy != "lg_accuracy" {
		t.Fatalf("repository query sort_by = %q, want lg_accuracy", repository.query.SortBy)
	}
	if repository.query.SortDirection != "asc" {
		t.Fatalf("repository query sort_direction = %q, want asc", repository.query.SortDirection)
	}
	if repository.query.From.Location() != time.UTC {
		t.Fatalf("repository query from location = %s, want UTC", repository.query.From.Location())
	}
	if repository.query.To.Location() != time.UTC {
		t.Fatalf("repository query to location = %s, want UTC", repository.query.To.Location())
	}
	if page.Filters.From == nil || page.Filters.To == nil {
		t.Fatal("filters from/to = nil, want non-nil")
	}
	if page.Filters.From.Location() != time.UTC {
		t.Fatalf("filters from location = %s, want UTC", page.Filters.From.Location())
	}
	if page.Filters.To.Location() != time.UTC {
		t.Fatalf("filters to location = %s, want UTC", page.Filters.To.Location())
	}
}

func TestServiceListPlayerRankingRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		query model.RankingQuery
	}{
		{
			name: "unsupported sort_by",
			query: model.RankingQuery{
				SortBy: "kills",
			},
		},
		{
			name: "unsupported sort_direction",
			query: model.RankingQuery{
				SortDirection: "sideways",
			},
		},
		{
			name: "negative limit",
			query: model.RankingQuery{
				Limit: -1,
			},
		},
		{
			name: "limit above max",
			query: model.RankingQuery{
				Limit: 101,
			},
		},
		{
			name: "negative offset",
			query: model.RankingQuery{
				Offset: -1,
			},
		},
		{
			name: "negative minimum matches",
			query: model.RankingQuery{
				MinimumMatches: -1,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repository := &stubRankingRepository{}
			svc := newTestService(t, repository)

			_, err := svc.ListPlayerRanking(context.Background(), tc.query)
			if err == nil {
				t.Fatal("ListPlayerRanking() error = nil, want non-nil")
			}
			if repository.called {
				t.Fatal("repository called = true, want false")
			}
		})
	}
}

func TestServiceListPlayerRankingBuildsStableMetaAndFilters(t *testing.T) {
	t.Parallel()

	repository := &stubRankingRepository{
		page: model.RankingPage{
			Data: []model.PlayerRankingRow{
				{PlayerID: "player-1", DisplayName: "Player One", Rank: 1},
				{PlayerID: "player-2", DisplayName: "Player Two", Rank: 2},
			},
			Meta: model.PaginationMeta{
				SortBy:         "ignored",
				SortDirection:  "ignored",
				MinimumMatches: 999,
				Limit:          999,
				Offset:         999,
				Returned:       999,
				HasNext:        true,
			},
			Filters: model.AppliedFilters{
				Mode: "ignored",
			},
		},
	}
	svc := newTestService(t, repository)

	page, err := svc.ListPlayerRanking(context.Background(), model.RankingQuery{
		Mode:           "1on1",
		Map:            "dm6",
		Server:         "server-a",
		SortBy:         "frags",
		SortDirection:  "desc",
		Limit:          2,
		Offset:         4,
		MinimumMatches: 15,
	})
	if err != nil {
		t.Fatalf("ListPlayerRanking() error = %v", err)
	}

	if len(page.Data) != 2 {
		t.Fatalf("data len = %d, want 2", len(page.Data))
	}
	if page.Meta.SortBy != "frags" {
		t.Fatalf("meta sort_by = %q, want frags", page.Meta.SortBy)
	}
	if page.Meta.SortDirection != "desc" {
		t.Fatalf("meta sort_direction = %q, want desc", page.Meta.SortDirection)
	}
	if page.Meta.MinimumMatches != 15 {
		t.Fatalf("meta minimum_matches = %d, want 15", page.Meta.MinimumMatches)
	}
	if page.Meta.Limit != 2 {
		t.Fatalf("meta limit = %d, want 2", page.Meta.Limit)
	}
	if page.Meta.Offset != 4 {
		t.Fatalf("meta offset = %d, want 4", page.Meta.Offset)
	}
	if page.Meta.Returned != 2 {
		t.Fatalf("meta returned = %d, want 2", page.Meta.Returned)
	}
	if !page.Meta.HasNext {
		t.Fatal("meta has_next = false, want true")
	}
	if page.Filters.Mode != "1on1" {
		t.Fatalf("filters mode = %q, want 1on1", page.Filters.Mode)
	}
	if page.Filters.Map != "dm6" {
		t.Fatalf("filters map = %q, want dm6", page.Filters.Map)
	}
	if page.Filters.Server != "server-a" {
		t.Fatalf("filters server = %q, want server-a", page.Filters.Server)
	}
}

func TestNewRankingServiceRequiresValidConfig(t *testing.T) {
	t.Parallel()

	_, err := NewRankingService(&stubRankingRepository{}, RankingConfig{})
	if err == nil {
		t.Fatal("NewRankingService() error = nil, want non-nil")
	}
}

func TestServiceListPlayerRankingPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	repository := &stubRankingRepository{err: errors.New("boom")}
	svc := newTestService(t, repository)

	_, err := svc.ListPlayerRanking(context.Background(), model.RankingQuery{})
	if err == nil {
		t.Fatal("ListPlayerRanking() error = nil, want non-nil")
	}
	if !repository.called {
		t.Fatal("repository called = false, want true")
	}
}

type stubRankingRepository struct {
	called bool
	query  model.RankingQuery
	page   model.RankingPage
	err    error
}

func (s *stubRankingRepository) ListPlayerRanking(_ context.Context, query model.RankingQuery) (model.RankingPage, error) {
	s.called = true
	s.query = query
	if s.err != nil {
		return model.RankingPage{}, s.err
	}
	return s.page, nil
}

func newTestService(t *testing.T, repository RankingRepository) *Service {
	t.Helper()

	svc, err := NewRankingService(repository, RankingConfig{
		DefaultLimit:   50,
		MaxLimit:       100,
		MinimumMatches: 10,
		DefaultSortBy:  model.DefaultSortBy,
		DefaultSortDir: model.DefaultSortDirection,
	})
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}

	return svc
}
