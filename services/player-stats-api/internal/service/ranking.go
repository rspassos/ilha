package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rspassos/ilha/services/player-stats-api/internal/model"
)

var (
	ErrInvalidLimit          = errors.New("query limit must be greater than or equal to zero")
	ErrInvalidOffset         = errors.New("query offset must be greater than or equal to zero")
	ErrInvalidMinimumMatches = errors.New("query minimum_matches must be greater than or equal to zero")
)

var allowedSortBy = map[string]struct{}{
	"efficiency":  {},
	"frags":       {},
	"lg_accuracy": {},
	"rl_hits":     {},
}

var allowedSortDirections = map[string]struct{}{
	"asc":  {},
	"desc": {},
}

type RankingService interface {
	ListPlayerRanking(ctx context.Context, query model.RankingQuery) (model.RankingPage, error)
}

type RankingRepository interface {
	ListPlayerRanking(ctx context.Context, query model.RankingQuery) (model.RankingPage, error)
}

type RankingServiceFunc func(ctx context.Context, query model.RankingQuery) (model.RankingPage, error)

type RankingConfig struct {
	DefaultLimit   int
	MaxLimit       int
	MinimumMatches int
	DefaultSortBy  string
	DefaultSortDir string
}

type Service struct {
	repository RankingRepository
	config     RankingConfig
}

func NewRankingService(repository RankingRepository, config RankingConfig) (*Service, error) {
	if repository == nil {
		return nil, errors.New("ranking repository must not be nil")
	}
	if config.DefaultLimit <= 0 {
		return nil, errors.New("ranking default_limit must be greater than zero")
	}
	if config.MaxLimit <= 0 {
		return nil, errors.New("ranking max_limit must be greater than zero")
	}
	if config.DefaultLimit > config.MaxLimit {
		return nil, errors.New("ranking default_limit must be less than or equal to max_limit")
	}
	if config.MinimumMatches <= 0 {
		return nil, errors.New("ranking minimum_matches must be greater than zero")
	}

	config.DefaultSortBy = strings.ToLower(strings.TrimSpace(config.DefaultSortBy))
	config.DefaultSortDir = strings.ToLower(strings.TrimSpace(config.DefaultSortDir))
	if config.DefaultSortBy == "" {
		config.DefaultSortBy = model.DefaultSortBy
	}
	if config.DefaultSortDir == "" {
		config.DefaultSortDir = model.DefaultSortDirection
	}
	if _, ok := allowedSortBy[config.DefaultSortBy]; !ok {
		return nil, fmt.Errorf("unsupported default sort_by %q", config.DefaultSortBy)
	}
	if _, ok := allowedSortDirections[config.DefaultSortDir]; !ok {
		return nil, fmt.Errorf("unsupported default sort_direction %q", config.DefaultSortDir)
	}

	return &Service{
		repository: repository,
		config:     config,
	}, nil
}

func (s *Service) ListPlayerRanking(ctx context.Context, query model.RankingQuery) (model.RankingPage, error) {
	normalized, err := s.normalizeQuery(query)
	if err != nil {
		return model.RankingPage{}, err
	}

	page, err := s.repository.ListPlayerRanking(ctx, normalized)
	if err != nil {
		return model.RankingPage{}, err
	}

	return model.NewRankingPage(normalized, page.Data, page.Meta.HasNext), nil
}

func (f RankingServiceFunc) ListPlayerRanking(ctx context.Context, query model.RankingQuery) (model.RankingPage, error) {
	return f(ctx, query)
}

func (s *Service) normalizeQuery(query model.RankingQuery) (model.RankingQuery, error) {
	query.Mode = strings.ToLower(strings.TrimSpace(query.Mode))
	query.Map = strings.TrimSpace(query.Map)
	query.Server = strings.TrimSpace(query.Server)
	query.SortBy = strings.ToLower(strings.TrimSpace(query.SortBy))
	query.SortDirection = strings.ToLower(strings.TrimSpace(query.SortDirection))

	if query.SortBy == "" {
		query.SortBy = s.config.DefaultSortBy
	}
	if query.SortDirection == "" {
		query.SortDirection = s.config.DefaultSortDir
	}

	if _, ok := allowedSortBy[query.SortBy]; !ok {
		return model.RankingQuery{}, fmt.Errorf("unsupported sort_by %q", query.SortBy)
	}
	if _, ok := allowedSortDirections[query.SortDirection]; !ok {
		return model.RankingQuery{}, fmt.Errorf("unsupported sort_direction %q", query.SortDirection)
	}

	switch {
	case query.Limit < 0:
		return model.RankingQuery{}, ErrInvalidLimit
	case query.Limit == 0:
		query.Limit = s.config.DefaultLimit
	case query.Limit > s.config.MaxLimit:
		return model.RankingQuery{}, fmt.Errorf("query limit must be less than or equal to %d", s.config.MaxLimit)
	}

	if query.Offset < 0 {
		return model.RankingQuery{}, ErrInvalidOffset
	}

	switch {
	case query.MinimumMatches < 0:
		return model.RankingQuery{}, ErrInvalidMinimumMatches
	case query.MinimumMatches == 0:
		query.MinimumMatches = s.config.MinimumMatches
	}

	if !query.From.IsZero() {
		query.From = query.From.UTC()
	}
	if !query.To.IsZero() {
		query.To = query.To.UTC()
	}

	return query, nil
}
