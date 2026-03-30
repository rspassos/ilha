package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rspassos/ilha/jobs/player-stats/internal/logging"
	"github.com/rspassos/ilha/jobs/player-stats/internal/metrics"
	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
	"github.com/rspassos/ilha/jobs/player-stats/internal/normalize"
	"github.com/rspassos/ilha/jobs/player-stats/internal/source"
	"github.com/rspassos/ilha/jobs/player-stats/internal/storage"
)

type Service struct {
	logger     *logging.Logger
	metrics    metricsRecorder
	repository analyticsRepository
	transform  rowTransformer
	source     source.MatchSource
	batchSize  int
	jobName    string
}

type analyticsRepository interface {
	LoadCheckpoint(context.Context, string) (model.Checkpoint, error)
	UpsertBatch(context.Context, model.ConsolidationBatch) (model.BatchResult, error)
}

type metricsRecorder interface {
	RecordMatchesScanned(string, int)
	RecordPlayerRowsUpserted(string, int)
	RecordIdentityResolution(string, int)
	RecordSkippedMatch(string)
	ObserveStage(string, time.Time)
}

type rowTransformer interface {
	BuildRows(context.Context, model.SourceMatch) ([]model.PlayerMatchRow, error)
}

func New(
	logger *logging.Logger,
	metricsCollector *metrics.Collector,
	repository *storage.Repository,
	matchSource source.MatchSource,
	jobName string,
	batchSize int,
) *Service {
	var analyticsRepo analyticsRepository
	if repository != nil {
		analyticsRepo = repository
	}
	var recorder metricsRecorder
	if metricsCollector != nil {
		recorder = metricsCollector
	}

	return &Service{
		logger:     logger,
		metrics:    recorder,
		repository: analyticsRepo,
		transform:  normalize.NewTransformer(),
		source:     matchSource,
		jobName:    jobName,
		batchSize:  batchSize,
	}
}

func (s *Service) ConsolidateMatch(ctx context.Context, match model.SourceMatch) (model.ConsolidationBatch, error) {
	if s == nil {
		return model.ConsolidationBatch{}, nil
	}
	if err := ctx.Err(); err != nil {
		return model.ConsolidationBatch{}, err
	}
	if s.transform == nil {
		return model.ConsolidationBatch{}, fmt.Errorf("transformer is not initialized")
	}

	rows, err := s.transform.BuildRows(ctx, match)
	if err != nil {
		return model.ConsolidationBatch{}, fmt.Errorf("build player rows for match %d: %w", match.CollectorMatchID, err)
	}

	return model.ConsolidationBatch{
		Rows: rows,
		Checkpoint: model.Checkpoint{
			JobName:              s.jobName,
			LastCollectorMatchID: match.CollectorMatchID,
		},
	}, nil
}

func (s *Service) RunOnce(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.repository == nil {
		return fmt.Errorf("repository is not initialized")
	}
	if s.source == nil {
		return fmt.Errorf("source is not initialized")
	}
	if s.batchSize <= 0 {
		return fmt.Errorf("batch size must be greater than zero")
	}

	runStartedAt := time.Now()
	checkpoint, err := s.repository.LoadCheckpoint(ctx, s.jobName)
	if err != nil {
		return fmt.Errorf("load checkpoint: %w", err)
	}
	if s.logger != nil {
		if err := s.logger.Info("player stats cycle started", map[string]any{
			"job_name":              s.jobName,
			"batch_size":            s.batchSize,
			"last_collector_match":  checkpoint.LastCollectorMatchID,
			"checkpoint_updated_at": checkpoint.UpdatedAt,
		}); err != nil {
			return err
		}
	}

	cursor := model.Cursor{LastCollectorMatchID: checkpoint.LastCollectorMatchID}
	summary := runSummary{
		LastCheckpoint: checkpoint.LastCollectorMatchID,
	}
	var runErr error

	for {
		listStartedAt := time.Now()
		matches, nextCursor, err := s.source.ListMatchesForConsolidation(ctx, cursor, s.batchSize)
		if s.metrics != nil {
			s.metrics.ObserveStage("list_matches", listStartedAt)
		}
		if err != nil {
			return fmt.Errorf("list matches for consolidation: %w", err)
		}
		if len(matches) == 0 {
			break
		}

		batchSummary, err := s.processBatch(ctx, matches, summary.LastCheckpoint)
		summary = summary.add(batchSummary)
		if err != nil {
			runErr = err
			break
		}

		if batchSummary.LastCheckpoint == summary.LastCheckpoint && nextCursor.LastCollectorMatchID == cursor.LastCollectorMatchID {
			break
		}
		cursor = model.Cursor{LastCollectorMatchID: batchSummary.LastCheckpoint}

		if batchSummary.HadFailures() {
			runErr = errors.Join(runErr, fmt.Errorf("player stats cycle stopped after batch failures"))
			break
		}
	}

	if s.metrics != nil {
		s.metrics.ObserveStage("run_once", runStartedAt)
	}
	if s.logger != nil {
		if err := s.logger.Info("player stats cycle completed", map[string]any{
			"job_name":                  s.jobName,
			"matches_scanned":           summary.MatchesScanned,
			"eligible_matches":          summary.EligibleMatches,
			"consolidated_matches":      summary.ConsolidatedMatches,
			"failed_matches":            summary.FailedMatches,
			"skipped_matches":           summary.SkippedMatches,
			"player_rows_persisted":     summary.PlayerRowsPersisted,
			"last_collector_match":      checkpoint.LastCollectorMatchID,
			"persisted_collector_match": summary.LastCheckpoint,
			"duration_seconds":          time.Since(runStartedAt).Seconds(),
		}); err != nil {
			return err
		}
	}

	if runErr != nil {
		return runErr
	}
	return nil
}

type runSummary struct {
	MatchesScanned      int
	EligibleMatches     int
	ConsolidatedMatches int
	FailedMatches       int
	SkippedMatches      int
	PlayerRowsPersisted int
	LastCheckpoint      int64
}

func (s *Service) processBatch(ctx context.Context, matches []model.SourceMatch, lastCheckpoint int64) (runSummary, error) {
	batchStartedAt := time.Now()
	summary := runSummary{
		MatchesScanned: len(matches),
		LastCheckpoint: lastCheckpoint,
	}

	rows := make([]model.PlayerMatchRow, 0)
	for _, match := range matches {
		if !match.Eligible() {
			summary.SkippedMatches++
			if summary.FailedMatches == 0 {
				summary.LastCheckpoint = match.CollectorMatchID
			}
			if s.metrics != nil {
				s.metrics.RecordSkippedMatch(match.SkipReason)
			}
			continue
		}

		summary.EligibleMatches++
		consolidateStartedAt := time.Now()
		batch, err := s.ConsolidateMatch(ctx, match)
		if s.metrics != nil {
			s.metrics.ObserveStage("consolidate_match", consolidateStartedAt)
		}
		if err != nil {
			summary.FailedMatches++
			if s.logger != nil {
				if logErr := s.logger.Error("player stats match consolidation failed", map[string]any{
					"collector_match_id": match.CollectorMatchID,
					"server_key":         match.ServerKey,
					"demo_name":          match.DemoName,
					"error":              err.Error(),
				}); logErr != nil {
					return summary, logErr
				}
			}
			continue
		}

		rows = append(rows, batch.Rows...)
		summary.ConsolidatedMatches++
		if summary.FailedMatches == 0 {
			summary.LastCheckpoint = match.CollectorMatchID
		}
	}

	s.recordBatchMetrics(summary)

	if summary.LastCheckpoint == lastCheckpoint && len(rows) == 0 {
		if s.logger != nil {
			if err := s.logger.Info("player stats batch scanned", map[string]any{
				"job_name":                  s.jobName,
				"matches_scanned":           summary.MatchesScanned,
				"eligible_matches":          summary.EligibleMatches,
				"consolidated_matches":      summary.ConsolidatedMatches,
				"failed_matches":            summary.FailedMatches,
				"skipped_matches":           summary.SkippedMatches,
				"persisted_collector_match": summary.LastCheckpoint,
				"checkpoint_persisted":      false,
				"duration_seconds":          time.Since(batchStartedAt).Seconds(),
			}); err != nil {
				return summary, err
			}
		}
		return summary, nil
	}

	persistStartedAt := time.Now()
	result, err := s.repository.UpsertBatch(ctx, model.ConsolidationBatch{
		Rows: rows,
		Checkpoint: model.Checkpoint{
			JobName:              s.jobName,
			LastCollectorMatchID: summary.LastCheckpoint,
		},
	})
	if s.metrics != nil {
		s.metrics.ObserveStage("persist_batch", persistStartedAt)
	}
	if err != nil {
		return summary, fmt.Errorf("upsert batch through collector_match_id %d: %w", summary.LastCheckpoint, err)
	}

	summary.PlayerRowsPersisted = result.StatsInserted + result.StatsUpdated
	s.recordPersistenceMetrics(result)

	if s.logger != nil {
		if err := s.logger.Info("player stats batch persisted", map[string]any{
			"job_name":                  s.jobName,
			"matches_scanned":           summary.MatchesScanned,
			"eligible_matches":          summary.EligibleMatches,
			"consolidated_matches":      summary.ConsolidatedMatches,
			"failed_matches":            summary.FailedMatches,
			"skipped_matches":           summary.SkippedMatches,
			"player_rows":               len(rows),
			"canonical_inserted":        result.CanonicalInserted,
			"canonical_reused":          result.CanonicalReused,
			"aliases_inserted":          result.AliasesInserted,
			"aliases_updated":           result.AliasesUpdated,
			"stats_inserted":            result.StatsInserted,
			"stats_updated":             result.StatsUpdated,
			"persisted_collector_match": summary.LastCheckpoint,
			"duration_seconds":          time.Since(batchStartedAt).Seconds(),
		}); err != nil {
			return summary, err
		}
	}

	return summary, nil
}

func (s *Service) recordBatchMetrics(summary runSummary) {
	if s.metrics == nil {
		return
	}
	if summary.EligibleMatches > 0 {
		s.metrics.RecordMatchesScanned("eligible", summary.EligibleMatches)
	}
	if summary.ConsolidatedMatches > 0 {
		s.metrics.RecordMatchesScanned("consolidated", summary.ConsolidatedMatches)
	}
	if summary.FailedMatches > 0 {
		s.metrics.RecordMatchesScanned("failed", summary.FailedMatches)
	}
	if summary.SkippedMatches > 0 {
		s.metrics.RecordMatchesScanned("skipped", summary.SkippedMatches)
	}
}

func (s *Service) recordPersistenceMetrics(result model.BatchResult) {
	if s.metrics == nil {
		return
	}
	if result.StatsInserted > 0 {
		s.metrics.RecordPlayerRowsUpserted("inserted", result.StatsInserted)
	}
	if result.StatsUpdated > 0 {
		s.metrics.RecordPlayerRowsUpserted("updated", result.StatsUpdated)
	}
	if result.CanonicalInserted > 0 {
		s.metrics.RecordIdentityResolution("created", result.CanonicalInserted)
	}
	if result.CanonicalReused > 0 {
		s.metrics.RecordIdentityResolution("reused", result.CanonicalReused)
	}
}

func (r runSummary) add(other runSummary) runSummary {
	r.MatchesScanned += other.MatchesScanned
	r.EligibleMatches += other.EligibleMatches
	r.ConsolidatedMatches += other.ConsolidatedMatches
	r.FailedMatches += other.FailedMatches
	r.SkippedMatches += other.SkippedMatches
	r.PlayerRowsPersisted += other.PlayerRowsPersisted
	if other.LastCheckpoint > r.LastCheckpoint {
		r.LastCheckpoint = other.LastCheckpoint
	}
	return r
}

func (r runSummary) HadFailures() bool {
	return r.FailedMatches > 0
}
