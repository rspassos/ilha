package merge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rspassos/ilha/jobs/collector/internal/config"
	"github.com/rspassos/ilha/jobs/collector/internal/model"
)

func TestMergeCombinesScoreAndStatsByDemo(t *testing.T) {
	t.Parallel()

	scores, stats := loadRealFixtures(t)
	service := New()

	records, warnings, err := service.Merge(testServerConfig(), scores, stats)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if len(records) != len(scores) {
		t.Fatalf("len(records) = %d, want %d", len(records), len(scores))
	}
	if len(warnings) != 7 {
		t.Fatalf("len(warnings) = %d, want 7", len(warnings))
	}

	duel := findRecord(t, records, "duel_expert_vs_matuzah[aerowalk]20260319-2013.mvd")
	if duel.MatchKey != "qlash-br-1:duel_expert_vs_matuzah[aerowalk]20260319-2013.mvd" {
		t.Fatalf("duel.MatchKey = %q", duel.MatchKey)
	}
	if duel.Mode != "duel" {
		t.Fatalf("duel.Mode = %q", duel.Mode)
	}
	if duel.MapName != "aerowalk" {
		t.Fatalf("duel.MapName = %q", duel.MapName)
	}
	if duel.Participants != "expert vs MatuzaH" {
		t.Fatalf("duel.Participants = %q", duel.Participants)
	}
	if duel.DurationSeconds != 600 {
		t.Fatalf("duel.DurationSeconds = %d, want 600", duel.DurationSeconds)
	}
	if duel.Hostname == "" {
		t.Fatal("duel.Hostname is empty")
	}
	if len(duel.ScorePayload) == 0 || len(duel.StatsPayload) == 0 || len(duel.MergedPayload) == 0 {
		t.Fatal("duel payloads should all be present")
	}

	team := findRecord(t, records, "2on2_red_vs_fk[dm4]20260319-2038.mvd")
	if team.Mode != "2on2" {
		t.Fatalf("team.Mode = %q, want 2on2", team.Mode)
	}
	if team.MapName != "dm4" {
		t.Fatalf("team.MapName = %q", team.MapName)
	}
	if team.Participants != "red vs fk" {
		t.Fatalf("team.Participants = %q", team.Participants)
	}
}

func TestMergeHandlesPartialPayloads(t *testing.T) {
	t.Parallel()

	service := New()
	server := testServerConfig()
	now := time.Date(2026, 3, 19, 20, 23, 28, 0, time.FixedZone("-0300", -3*3600))

	scoreOnly := model.ScoreMatch{
		Demo:         "score-only.mvd",
		TimestampISO: "2026-03-19T20:23:28-03:00",
		PlayedAt:     now,
		Mode:         "duel",
		Map:          "dm6",
		Participants: "alpha vs beta",
	}
	statsOnly := model.StatsMatch{
		Demo:     "stats-only.mvd",
		Date:     "2026-03-19 20:23:28 -0300",
		PlayedAt: now,
		Mode:     "team",
		Map:      "dm2",
		Teams:    []string{"red", "blue"},
		Duration: 600,
		Hostname: "test-host",
	}

	records, warnings, err := service.Merge(server, []model.ScoreMatch{scoreOnly}, []model.StatsMatch{statsOnly})
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}
	if !containsWarning(warnings, WarningScoreOnly) {
		t.Fatalf("warnings = %#v, want %q", warnings, WarningScoreOnly)
	}
	if !containsWarning(warnings, WarningStatsOnly) {
		t.Fatalf("warnings = %#v, want %q", warnings, WarningStatsOnly)
	}

	scoreOnlyRecord := findRecord(t, records, "score-only.mvd")
	if len(scoreOnlyRecord.ScorePayload) == 0 {
		t.Fatal("scoreOnlyRecord.ScorePayload is empty")
	}
	if scoreOnlyRecord.StatsPayload != nil {
		t.Fatalf("scoreOnlyRecord.StatsPayload = %s, want nil", scoreOnlyRecord.StatsPayload)
	}

	statsOnlyRecord := findRecord(t, records, "stats-only.mvd")
	if statsOnlyRecord.Participants != "red vs blue" {
		t.Fatalf("statsOnlyRecord.Participants = %q", statsOnlyRecord.Participants)
	}
	if statsOnlyRecord.DurationSeconds != 600 {
		t.Fatalf("statsOnlyRecord.DurationSeconds = %d", statsOnlyRecord.DurationSeconds)
	}
}

func TestMergeDerivesHasBotsFromNestedScorePlayers(t *testing.T) {
	t.Parallel()

	service := New()
	server := testServerConfig()
	now := time.Date(2026, 3, 19, 20, 48, 44, 0, time.FixedZone("-0300", -3*3600))

	records, warnings, err := service.Merge(server, []model.ScoreMatch{
		{
			Demo:         "bots.mvd",
			TimestampISO: "2026-03-19T20:48:44-03:00",
			PlayedAt:     now,
			Mode:         "2on2",
			Map:          "dm4",
			Teams: []model.ScoreTeam{
				{
					Name: "red",
					Players: []model.ScorePlayer{
						{Name: "bot-one", IsBot: true},
					},
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if len(warnings) != 1 || warnings[0].Reason != WarningScoreOnly {
		t.Fatalf("warnings = %#v", warnings)
	}
	if !records[0].HasBots {
		t.Fatal("records[0].HasBots = false, want true")
	}
}

func TestMergeWarnsAndDeduplicatesRepeatedDemos(t *testing.T) {
	t.Parallel()

	service := New()
	server := testServerConfig()
	now := time.Date(2026, 3, 19, 20, 23, 28, 0, time.FixedZone("-0300", -3*3600))

	scoreA := model.ScoreMatch{
		Demo:         "dup.mvd",
		TimestampISO: "2026-03-19T20:23:28-03:00",
		PlayedAt:     now,
		Mode:         "duel",
		Map:          "dm6",
		Participants: "one vs two",
	}
	scoreB := scoreA
	scoreB.Map = "aerowalk"

	statA := model.StatsMatch{
		Demo:     "dup.mvd",
		Date:     "2026-03-19 20:23:28 -0300",
		PlayedAt: now,
		Mode:     "duel",
		Map:      "aerowalk",
	}
	statB := statA
	statB.Hostname = "second"

	records, warnings, err := service.Merge(server, []model.ScoreMatch{scoreA, scoreB}, []model.StatsMatch{statA, statB})
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if !containsWarning(warnings, WarningDuplicateScores) {
		t.Fatalf("warnings = %#v, want duplicate score warning", warnings)
	}
	if !containsWarning(warnings, WarningDuplicateStats) {
		t.Fatalf("warnings = %#v, want duplicate stats warning", warnings)
	}
	if records[0].MapName != "aerowalk" {
		t.Fatalf("records[0].MapName = %q, want aerowalk", records[0].MapName)
	}
	if records[0].Hostname != "second" {
		t.Fatalf("records[0].Hostname = %q, want second", records[0].Hostname)
	}
}

func TestMergeWarnsAboutMismatchedFields(t *testing.T) {
	t.Parallel()

	service := New()
	server := testServerConfig()
	scoreTime := time.Date(2026, 3, 19, 20, 23, 28, 0, time.FixedZone("-0300", -3*3600))
	statsTime := scoreTime.Add(time.Minute)

	_, warnings, err := service.Merge(server, []model.ScoreMatch{
		{
			Demo:         "mismatch.mvd",
			TimestampISO: "2026-03-19T20:23:28-03:00",
			PlayedAt:     scoreTime,
			Mode:         "2on2",
			Map:          "dm4",
			Participants: "red vs blue",
		},
	}, []model.StatsMatch{
		{
			Demo:     "mismatch.mvd",
			Date:     "2026-03-19 20:24:28 -0300",
			PlayedAt: statsTime,
			Mode:     "team",
			Map:      "aerowalk",
		},
	})
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	for _, reason := range []string{WarningModeMismatch, WarningMapMismatch, WarningPlayedAtMismatch} {
		if !containsWarning(warnings, reason) {
			t.Fatalf("warnings = %#v, want %q", warnings, reason)
		}
	}
}

func loadRealFixtures(t *testing.T) ([]model.ScoreMatch, []model.StatsMatch) {
	t.Helper()

	var scores []model.ScoreMatch
	var stats []model.StatsMatch

	scoreData := mustReadRepoFile(t, "../../../../docs/responses/lastscores.json")
	if err := json.Unmarshal(scoreData, &scores); err != nil {
		t.Fatalf("unmarshal lastscores fixture: %v", err)
	}
	for index := range scores {
		if err := scores[index].Normalize(); err != nil {
			t.Fatalf("normalize score fixture %d: %v", index, err)
		}
	}

	statsData := mustReadRepoFile(t, "../../../../docs/responses/laststats.json")
	if err := json.Unmarshal(statsData, &stats); err != nil {
		t.Fatalf("unmarshal laststats fixture: %v", err)
	}
	for index := range stats {
		if err := stats[index].Normalize(); err != nil {
			t.Fatalf("normalize stats fixture %d: %v", index, err)
		}
	}

	return scores, stats
}

func testServerConfig() config.ServerConfig {
	return config.ServerConfig{
		Key:            "qlash-br-1",
		Name:           "Qlash Brazil 1",
		Address:        "qw.qlash.com.br:28501",
		Enabled:        true,
		TimeoutSeconds: 5,
	}
}

func findRecord(t *testing.T, records []model.MatchRecord, demo string) model.MatchRecord {
	t.Helper()
	for _, record := range records {
		if record.DemoName == demo {
			return record
		}
	}
	t.Fatalf("record %q not found", demo)
	return model.MatchRecord{}
}

func containsWarning(warnings []model.MergeWarning, reason string) bool {
	for _, warning := range warnings {
		if warning.Reason == reason {
			return true
		}
	}
	return false
}

func mustReadRepoFile(t *testing.T, relativePath string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(relativePath))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}
	return data
}
