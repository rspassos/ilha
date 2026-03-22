package collector

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rspassos/ilha/jobs/collector/internal/config"
	"github.com/rspassos/ilha/jobs/collector/internal/logging"
	"github.com/rspassos/ilha/jobs/collector/internal/metrics"
	"github.com/rspassos/ilha/jobs/collector/internal/model"
	"github.com/rspassos/ilha/jobs/collector/internal/storage"
)

func TestServiceCollectServerRunsFullFlow(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := logging.New(&logBuffer, "match-stats-collector")
	service := NewService(
		fakeScoresClient{matches: []model.ScoreMatch{{Demo: "demo.mvd"}}},
		fakeStatsClient{matches: []model.StatsMatch{{Demo: "demo.mvd"}}},
		&fakeRepository{result: storage.UpsertResult{Inserted: 1}},
		fakeMerger{records: []model.MatchRecord{{DemoName: "demo.mvd"}}, warnings: []model.MergeWarning{{ServerKey: "qlash-br-1", DemoName: "demo.mvd", Reason: "stats_only"}}},
		logger,
		metrics.New(),
	)
	service.now = func() time.Time { return time.Now().Add(-time.Second) }

	result, err := service.CollectServer(context.Background(), testServer())
	if err != nil {
		t.Fatalf("CollectServer() error = %v", err)
	}

	if result.Inserted != 1 {
		t.Fatalf("result.Inserted = %d, want 1", result.Inserted)
	}
	if !strings.Contains(logBuffer.String(), `"reason":"stats_only"`) {
		t.Fatalf("logs missing warning event:\n%s", logBuffer.String())
	}
}

func TestServiceRunOnceContinuesAfterServerFailure(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	service := NewService(
		fakeScoresClient{matches: []model.ScoreMatch{{Demo: "demo.mvd"}}, errByServer: map[string]error{"server-b": errors.New("upstream down")}},
		fakeStatsClient{matches: []model.StatsMatch{{Demo: "demo.mvd"}}},
		&fakeRepository{result: storage.UpsertResult{Inserted: 1}},
		fakeMerger{records: []model.MatchRecord{{DemoName: "demo.mvd"}}},
		logging.New(&logBuffer, "match-stats-collector"),
		metrics.New(),
	)
	service.now = time.Now

	err := service.RunOnce(context.Background(), []config.ServerConfig{
		{Key: "server-a", Name: "A"},
		{Key: "server-b", Name: "B"},
	})
	if err == nil {
		t.Fatal("RunOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `fetch lastscores for server "server-b"`) {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if !strings.Contains(logBuffer.String(), `"server_key":"server-a"`) {
		t.Fatalf("logs missing successful server:\n%s", logBuffer.String())
	}
}

type fakeScoresClient struct {
	matches     []model.ScoreMatch
	errByServer map[string]error
}

func (f fakeScoresClient) FetchLastScores(_ context.Context, server config.ServerConfig) ([]model.ScoreMatch, error) {
	if err := f.errByServer[server.Key]; err != nil {
		return nil, err
	}
	return f.matches, nil
}

type fakeStatsClient struct {
	matches []model.StatsMatch
}

func (f fakeStatsClient) FetchLastStats(context.Context, config.ServerConfig) ([]model.StatsMatch, error) {
	return f.matches, nil
}

type fakeRepository struct {
	result storage.UpsertResult
	err    error
}

func (f *fakeRepository) UpsertMatches(context.Context, []model.MatchRecord) (storage.UpsertResult, error) {
	return f.result, f.err
}

type fakeMerger struct {
	records  []model.MatchRecord
	warnings []model.MergeWarning
	err      error
}

func (f fakeMerger) Merge(config.ServerConfig, []model.ScoreMatch, []model.StatsMatch) ([]model.MatchRecord, []model.MergeWarning, error) {
	return f.records, f.warnings, f.err
}

func testServer() config.ServerConfig {
	return config.ServerConfig{Key: "qlash-br-1", Name: "Qlash Brazil 1"}
}
