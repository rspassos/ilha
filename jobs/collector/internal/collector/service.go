package collector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rspassos/ilha/jobs/collector/internal/config"
	"github.com/rspassos/ilha/jobs/collector/internal/merge"
	"github.com/rspassos/ilha/jobs/collector/internal/metrics"
	"github.com/rspassos/ilha/jobs/collector/internal/model"
	"github.com/rspassos/ilha/jobs/collector/internal/storage"
)

type ScoresClient interface {
	FetchLastScores(ctx context.Context, server config.ServerConfig) ([]model.ScoreMatch, error)
}

type StatsClient interface {
	FetchLastStats(ctx context.Context, server config.ServerConfig) ([]model.StatsMatch, error)
}

type MatchRepository interface {
	UpsertMatches(ctx context.Context, matches []model.MatchRecord) (storage.UpsertResult, error)
}

type MatchMerger interface {
	Merge(server config.ServerConfig, scores []model.ScoreMatch, stats []model.StatsMatch) ([]model.MatchRecord, []model.MergeWarning, error)
}

type Logger interface {
	Info(message string, fields map[string]any) error
	Warn(message string, fields map[string]any) error
	Error(message string, fields map[string]any) error
}

type Service struct {
	scoresClient ScoresClient
	statsClient  StatsClient
	repository   MatchRepository
	merger       MatchMerger
	metrics      *metrics.Collector
	logger       Logger
	now          func() time.Time
}

type RunResult struct {
	Servers         int
	Succeeded       int
	Failed          int
	Warnings        int
	MatchesFetched  int
	MatchesUpserted int
}

type ServerRunResult struct {
	ServerKey     string
	ScoresFetched int
	StatsFetched  int
	Warnings      []model.MergeWarning
	Inserted      int
	Updated       int
}

func NewService(scoresClient ScoresClient, statsClient StatsClient, repository MatchRepository, merger MatchMerger, logger Logger, metricsCollector *metrics.Collector) *Service {
	if merger == nil {
		merger = merge.New()
	}
	return &Service{
		scoresClient: scoresClient,
		statsClient:  statsClient,
		repository:   repository,
		merger:       merger,
		logger:       logger,
		metrics:      metricsCollector,
		now:          time.Now().UTC,
	}
}

func (s *Service) RunOnce(ctx context.Context, servers []config.ServerConfig) error {
	result := RunResult{Servers: len(servers)}
	if s.metrics != nil {
		s.metrics.RecordRun("started")
	}

	var runErrors []error
	for _, server := range servers {
		serverResult, err := s.CollectServer(ctx, server)
		if err != nil {
			result.Failed++
			runErrors = append(runErrors, err)
			if s.metrics != nil {
				s.metrics.RecordServerRun(server.Key, "error")
			}
			_ = s.logger.Error("collector server run failed", map[string]any{
				"server_key":  server.Key,
				"server_name": server.Name,
				"error":       err.Error(),
			})
			continue
		}

		result.Succeeded++
		result.Warnings += len(serverResult.Warnings)
		result.MatchesFetched += serverResult.ScoresFetched + serverResult.StatsFetched
		result.MatchesUpserted += serverResult.Inserted + serverResult.Updated
		if s.metrics != nil {
			s.metrics.RecordServerRun(server.Key, "success")
		}
		_ = s.logger.Info("collector server run completed", map[string]any{
			"server_key":     server.Key,
			"server_name":    server.Name,
			"scores_fetched": serverResult.ScoresFetched,
			"stats_fetched":  serverResult.StatsFetched,
			"warnings":       len(serverResult.Warnings),
			"inserted":       serverResult.Inserted,
			"updated":        serverResult.Updated,
		})
	}

	status := "success"
	if len(runErrors) > 0 {
		status = "partial_failure"
	}
	if s.metrics != nil {
		s.metrics.RecordRun(status)
	}
	_ = s.logger.Info("collector run completed", map[string]any{
		"server_count":      result.Servers,
		"succeeded_servers": result.Succeeded,
		"failed_servers":    result.Failed,
		"warnings":          result.Warnings,
		"matches_fetched":   result.MatchesFetched,
		"matches_upserted":  result.MatchesUpserted,
		"status":            status,
	})

	if len(runErrors) > 0 {
		return errors.Join(runErrors...)
	}
	return nil
}

func (s *Service) CollectServer(ctx context.Context, server config.ServerConfig) (ServerRunResult, error) {
	if s.scoresClient == nil || s.statsClient == nil || s.repository == nil || s.merger == nil {
		return ServerRunResult{}, errors.New("collector dependencies are not initialized")
	}

	_ = s.logger.Info("collector server run started", map[string]any{
		"server_key":  server.Key,
		"server_name": server.Name,
	})

	scoreStartedAt := s.now()
	scores, err := s.scoresClient.FetchLastScores(ctx, server)
	if s.metrics != nil {
		s.metrics.ObserveRequest(server.Key, "lastscores", scoreStartedAt)
	}
	if err != nil {
		return ServerRunResult{}, fmt.Errorf("fetch lastscores for server %q: %w", server.Key, err)
	}
	if s.metrics != nil {
		s.metrics.RecordMatchesFetched(server.Key, "lastscores", len(scores))
	}

	statsStartedAt := s.now()
	stats, err := s.statsClient.FetchLastStats(ctx, server)
	if s.metrics != nil {
		s.metrics.ObserveRequest(server.Key, "laststats", statsStartedAt)
	}
	if err != nil {
		return ServerRunResult{}, fmt.Errorf("fetch laststats for server %q: %w", server.Key, err)
	}
	if s.metrics != nil {
		s.metrics.RecordMatchesFetched(server.Key, "laststats", len(stats))
	}

	records, warnings, err := s.merger.Merge(server, scores, stats)
	if err != nil {
		return ServerRunResult{}, fmt.Errorf("merge matches for server %q: %w", server.Key, err)
	}
	for _, warning := range warnings {
		if s.metrics != nil {
			s.metrics.RecordMergeWarning(server.Key, warning.Reason)
		}
		_ = s.logger.Warn("collector merge warning", map[string]any{
			"server_key": server.Key,
			"demo_name":  warning.DemoName,
			"reason":     warning.Reason,
		})
	}

	upsertResult, err := s.repository.UpsertMatches(ctx, records)
	if err != nil {
		return ServerRunResult{}, fmt.Errorf("persist matches for server %q: %w", server.Key, err)
	}
	if s.metrics != nil {
		s.metrics.RecordMatchesUpserted(server.Key, "inserted", upsertResult.Inserted)
		s.metrics.RecordMatchesUpserted(server.Key, "updated", upsertResult.Updated)
	}

	return ServerRunResult{
		ServerKey:     server.Key,
		ScoresFetched: len(scores),
		StatsFetched:  len(stats),
		Warnings:      warnings,
		Inserted:      upsertResult.Inserted,
		Updated:       upsertResult.Updated,
	}, nil
}
