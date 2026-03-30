package service

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rspassos/ilha/jobs/player-stats/internal/logging"
	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
)

func TestRunOnceRequiresDependencies(t *testing.T) {
	t.Parallel()

	service := New(nil, nil, nil, nil, "player-stats", 100)
	if err := service.RunOnce(context.Background()); err == nil || err.Error() != "repository is not initialized" {
		t.Fatalf("RunOnce() error = %v, want repository is not initialized", err)
	}
}

func TestRunOnceRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	service := &Service{}
	if err := service.RunOnce(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunOnce() error = %v, want context canceled", err)
	}
}

func TestRunOnceProcessesBatchAndAdvancesCheckpoint(t *testing.T) {
	t.Parallel()

	repository := &stubRepository{
		checkpoint: model.Checkpoint{
			JobName:              "player-stats",
			LastCollectorMatchID: 7,
		},
		upsertResults: []stubUpsertResult{
			{
				result: model.BatchResult{
					CanonicalInserted: 1,
					AliasesInserted:   1,
					StatsInserted:     2,
				},
			},
		},
	}
	source := &stubSource{
		pages: []stubSourcePage{
			{
				matches: []model.SourceMatch{
					{CollectorMatchID: 8, SkipReason: "missing_stats_payload"},
					{CollectorMatchID: 9, ServerKey: "qlash-br-1", DemoName: "demo-9.mvd"},
				},
			},
			{},
		},
	}
	metrics := &stubMetrics{}
	transform := stubTransformer{
		rowsByMatch: map[int64][]model.PlayerMatchRow{
			9: {
				{CollectorMatchID: 9, ObservedName: "Alpha"},
				{CollectorMatchID: 9, ObservedName: "Bravo"},
			},
		},
	}

	service := &Service{
		repository: repository,
		source:     source,
		metrics:    metrics,
		transform:  transform,
		jobName:    "player-stats",
		batchSize:  2,
	}

	if err := service.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(source.calls) != 2 {
		t.Fatalf("source calls = %d, want 2", len(source.calls))
	}
	if source.calls[0].cursor.LastCollectorMatchID != 7 {
		t.Fatalf("first source cursor = %d, want 7", source.calls[0].cursor.LastCollectorMatchID)
	}
	if source.calls[1].cursor.LastCollectorMatchID != 9 {
		t.Fatalf("second source cursor = %d, want 9", source.calls[1].cursor.LastCollectorMatchID)
	}
	if len(repository.upsertCalls) != 1 {
		t.Fatalf("upsert calls = %d, want 1", len(repository.upsertCalls))
	}
	call := repository.upsertCalls[0]
	if call.Checkpoint.LastCollectorMatchID != 9 {
		t.Fatalf("saved checkpoint id = %d, want 9", call.Checkpoint.LastCollectorMatchID)
	}
	if len(call.Rows) != 2 {
		t.Fatalf("persisted rows = %d, want 2", len(call.Rows))
	}
	if len(metrics.skippedReasons) != 1 || metrics.skippedReasons[0] != "missing_stats_payload" {
		t.Fatalf("skipped reasons = %#v, want [missing_stats_payload]", metrics.skippedReasons)
	}
	if got := metrics.sumMatches("eligible"); got != 1 {
		t.Fatalf("eligible matches metric = %d, want 1", got)
	}
	if got := metrics.sumMatches("consolidated"); got != 1 {
		t.Fatalf("consolidated matches metric = %d, want 1", got)
	}
	if got := metrics.sumMatches("skipped"); got != 1 {
		t.Fatalf("skipped matches metric = %d, want 1", got)
	}
	if got := metrics.sumRows("inserted"); got != 2 {
		t.Fatalf("inserted rows metric = %d, want 2", got)
	}
	if got := metrics.sumIdentity("created"); got != 1 {
		t.Fatalf("created identity metric = %d, want 1", got)
	}
}

func TestRunOncePersistsSuccessfulRowsButKeepsCheckpointBeforeFailedMatch(t *testing.T) {
	t.Parallel()

	repository := &stubRepository{
		checkpoint: model.Checkpoint{
			JobName:              "player-stats",
			LastCollectorMatchID: 10,
		},
		upsertResults: []stubUpsertResult{
			{
				result: model.BatchResult{
					CanonicalInserted: 1,
					CanonicalReused:   1,
					AliasesInserted:   1,
					AliasesUpdated:    1,
					StatsInserted:     1,
					StatsUpdated:      1,
				},
			},
		},
	}
	source := &stubSource{
		pages: []stubSourcePage{
			{
				matches: []model.SourceMatch{
					{CollectorMatchID: 11, ServerKey: "qlash-br-1", DemoName: "demo-11.mvd"},
					{CollectorMatchID: 12, ServerKey: "qlash-br-1", DemoName: "demo-12.mvd"},
					{CollectorMatchID: 13, ServerKey: "qlash-br-1", DemoName: "demo-13.mvd"},
				},
			},
		},
	}
	transform := stubTransformer{
		rowsByMatch: map[int64][]model.PlayerMatchRow{
			11: {{CollectorMatchID: 11, ObservedName: "Alpha"}},
			13: {{CollectorMatchID: 13, ObservedName: "Charlie"}},
		},
		errByMatch: map[int64]error{
			12: errors.New("unsupported active player count 3"),
		},
	}
	metrics := &stubMetrics{}
	var logs bytes.Buffer

	service := &Service{
		logger:     logging.New(&logs, "player-stats"),
		repository: repository,
		source:     source,
		metrics:    metrics,
		transform:  transform,
		jobName:    "player-stats",
		batchSize:  3,
	}

	err := service.RunOnce(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cycle stopped after batch failures") {
		t.Fatalf("RunOnce() error = %v, want batch failure summary", err)
	}
	if len(repository.upsertCalls) != 1 {
		t.Fatalf("upsert calls = %d, want 1", len(repository.upsertCalls))
	}
	call := repository.upsertCalls[0]
	if call.Checkpoint.LastCollectorMatchID != 11 {
		t.Fatalf("saved checkpoint id = %d, want 11", call.Checkpoint.LastCollectorMatchID)
	}
	if len(call.Rows) != 2 {
		t.Fatalf("persisted rows = %d, want 2", len(call.Rows))
	}
	if call.Rows[0].CollectorMatchID != 11 || call.Rows[1].CollectorMatchID != 13 {
		t.Fatalf("persisted row match ids = [%d %d], want [11 13]", call.Rows[0].CollectorMatchID, call.Rows[1].CollectorMatchID)
	}
	if got := metrics.sumMatches("failed"); got != 1 {
		t.Fatalf("failed matches metric = %d, want 1", got)
	}
	if !strings.Contains(logs.String(), `"message":"player stats match consolidation failed"`) {
		t.Fatalf("logs = %s, want consolidation failure entry", logs.String())
	}
}

func TestRunOnceReturnsPersistenceErrorWithoutAdvancingCheckpoint(t *testing.T) {
	t.Parallel()

	repository := &stubRepository{
		checkpoint: model.Checkpoint{
			JobName:              "player-stats",
			LastCollectorMatchID: 3,
		},
		upsertResults: []stubUpsertResult{
			{err: errors.New("database unavailable")},
		},
	}
	source := &stubSource{
		pages: []stubSourcePage{
			{
				matches: []model.SourceMatch{
					{CollectorMatchID: 4, ServerKey: "qlash-br-1", DemoName: "demo-4.mvd"},
				},
			},
		},
	}
	transform := stubTransformer{
		rowsByMatch: map[int64][]model.PlayerMatchRow{
			4: {{CollectorMatchID: 4, ObservedName: "Alpha"}},
		},
	}

	service := &Service{
		repository: repository,
		source:     source,
		transform:  transform,
		jobName:    "player-stats",
		batchSize:  1,
	}

	err := service.RunOnce(context.Background())
	if err == nil || !strings.Contains(err.Error(), "database unavailable") {
		t.Fatalf("RunOnce() error = %v, want database unavailable", err)
	}
	if len(repository.upsertCalls) != 1 {
		t.Fatalf("upsert calls = %d, want 1", len(repository.upsertCalls))
	}
	if repository.upsertCalls[0].Checkpoint.LastCollectorMatchID != 4 {
		t.Fatalf("upsert checkpoint id = %d, want 4", repository.upsertCalls[0].Checkpoint.LastCollectorMatchID)
	}
}

func TestConsolidateMatchBuildsCheckpointedRows(t *testing.T) {
	t.Parallel()

	service := &Service{
		transform: stubTransformer{
			rowsByMatch: map[int64][]model.PlayerMatchRow{
				9: {
					{CollectorMatchID: 9, ObservedName: "Alpha"},
					{CollectorMatchID: 9, ObservedName: "Bravo"},
				},
			},
		},
		jobName: "player-stats",
	}

	batch, err := service.ConsolidateMatch(context.Background(), model.SourceMatch{CollectorMatchID: 9})
	if err != nil {
		t.Fatalf("ConsolidateMatch() error = %v", err)
	}
	if len(batch.Rows) != 2 {
		t.Fatalf("len(batch.Rows) = %d, want 2", len(batch.Rows))
	}
	if batch.Checkpoint.JobName != "player-stats" {
		t.Fatalf("batch.Checkpoint.JobName = %q, want player-stats", batch.Checkpoint.JobName)
	}
	if batch.Checkpoint.LastCollectorMatchID != 9 {
		t.Fatalf("batch.Checkpoint.LastCollectorMatchID = %d, want 9", batch.Checkpoint.LastCollectorMatchID)
	}
}

func TestConsolidateMatchRequiresTransformer(t *testing.T) {
	t.Parallel()

	service := &Service{}
	_, err := service.ConsolidateMatch(context.Background(), model.SourceMatch{CollectorMatchID: 4})
	if err == nil || !strings.Contains(err.Error(), "transformer is not initialized") {
		t.Fatalf("ConsolidateMatch() error = %v, want transformer is not initialized", err)
	}
}

type stubUpsertResult struct {
	result model.BatchResult
	err    error
}

type stubRepository struct {
	checkpoint    model.Checkpoint
	upsertCalls   []model.ConsolidationBatch
	upsertResults []stubUpsertResult
}

func (r *stubRepository) LoadCheckpoint(context.Context, string) (model.Checkpoint, error) {
	return r.checkpoint, nil
}

func (r *stubRepository) UpsertBatch(_ context.Context, batch model.ConsolidationBatch) (model.BatchResult, error) {
	r.upsertCalls = append(r.upsertCalls, batch)
	if len(r.upsertResults) == 0 {
		return model.BatchResult{}, nil
	}
	result := r.upsertResults[0]
	r.upsertResults = r.upsertResults[1:]
	return result.result, result.err
}

type stubSourcePage struct {
	matches []model.SourceMatch
	err     error
}

type stubSource struct {
	pages []stubSourcePage
	calls []struct {
		cursor model.Cursor
		limit  int
	}
}

func (s *stubSource) ListMatchesForConsolidation(_ context.Context, cursor model.Cursor, limit int) ([]model.SourceMatch, model.Cursor, error) {
	s.calls = append(s.calls, struct {
		cursor model.Cursor
		limit  int
	}{cursor: cursor, limit: limit})
	if len(s.pages) == 0 {
		return nil, cursor, nil
	}
	page := s.pages[0]
	s.pages = s.pages[1:]
	nextCursor := cursor
	if len(page.matches) > 0 {
		nextCursor.LastCollectorMatchID = page.matches[len(page.matches)-1].CollectorMatchID
	}
	return page.matches, nextCursor, page.err
}

type stubMetrics struct {
	skippedReasons []string
	matchCounts    map[string]int
	rowCounts      map[string]int
	identityCounts map[string]int
	stageCalls     []string
}

type stubTransformer struct {
	rowsByMatch map[int64][]model.PlayerMatchRow
	errByMatch  map[int64]error
}

func (t stubTransformer) BuildRows(ctx context.Context, match model.SourceMatch) ([]model.PlayerMatchRow, error) {
	_ = ctx
	if err := t.errByMatch[match.CollectorMatchID]; err != nil {
		return nil, err
	}
	return t.rowsByMatch[match.CollectorMatchID], nil
}

func (m *stubMetrics) RecordSkippedMatch(reason string) {
	m.skippedReasons = append(m.skippedReasons, reason)
}

func (m *stubMetrics) RecordMatchesScanned(result string, count int) {
	if m.matchCounts == nil {
		m.matchCounts = make(map[string]int)
	}
	m.matchCounts[result] += count
}

func (m *stubMetrics) RecordPlayerRowsUpserted(result string, count int) {
	if m.rowCounts == nil {
		m.rowCounts = make(map[string]int)
	}
	m.rowCounts[result] += count
}

func (m *stubMetrics) RecordIdentityResolution(result string, count int) {
	if m.identityCounts == nil {
		m.identityCounts = make(map[string]int)
	}
	m.identityCounts[result] += count
}

func (m *stubMetrics) ObserveStage(stage string, startedAt time.Time) {
	_ = startedAt
	m.stageCalls = append(m.stageCalls, stage)
}

func (m *stubMetrics) sumMatches(label string) int {
	return m.matchCounts[label]
}

func (m *stubMetrics) sumRows(label string) int {
	return m.rowCounts[label]
}

func (m *stubMetrics) sumIdentity(label string) int {
	return m.identityCounts[label]
}
