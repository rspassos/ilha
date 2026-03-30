package normalize

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
)

func TestNormalizeMode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		dm            int
		activePlayers int
		want          string
		wantErr       bool
	}{
		{name: "dm4 overrides player count", dm: 4, activePlayers: 3, want: ModeDMM4},
		{name: "duel", dm: 3, activePlayers: 2, want: Mode1v1},
		{name: "2v2", dm: 3, activePlayers: 4, want: Mode2v2},
		{name: "3v3", dm: 3, activePlayers: 6, want: Mode3v3},
		{name: "4v4", dm: 3, activePlayers: 8, want: Mode4v4},
		{name: "unsupported", dm: 3, activePlayers: 5, wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeMode(tc.dm, tc.activePlayers)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizeMode() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeMode() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeMode() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildRowsExtractsMetricsAndDerivedValues(t *testing.T) {
	t.Parallel()

	transformer := NewTransformer()
	transformer.now = func() time.Time {
		return time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	}

	match := model.SourceMatch{
		CollectorMatchID: 44,
		ServerKey:        "qlash-br-1",
		DemoName:         "demo-1.mvd",
		MapName:          "dm6",
		RawMode:          "team",
		PlayedAt:         time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC),
		Stats: model.SourceStatsMatch{
			Demo: "demo-1.mvd",
			Map:  "dm6",
			Mode: "team",
			DM:   3,
			Players: []model.SourcePlayer{
				testPlayer(t, "Alpha", "alpha-login", "red", map[string]int{
					"frags":    35,
					"deaths":   12,
					"kills":    30,
					"tk":       1,
					"suicides": 2,
				}, map[string]int{
					"taken": 2500,
					"given": 3200,
				}, map[string]int{
					"max":  4,
					"quad": 1,
				}, map[string]any{
					"rl": map[string]any{
						"acc":   map[string]any{"hits": 18},
						"kills": map[string]any{"total": 10},
					},
					"lg": map[string]any{
						"acc": map[string]any{"attacks": 100, "hits": 42},
					},
				}, map[string]any{
					"ga":         map[string]any{"took": 2},
					"ra":         map[string]any{"took": 3},
					"ya":         map[string]any{"took": 5},
					"health_100": map[string]any{"took": 6},
				}, 22, model.SourcePlayer{}.Bot),
				testPlayer(t, "Bravo", "", "blue", map[string]int{
					"frags":  20,
					"deaths": 18,
					"kills":  20,
				}, nil, nil, nil, nil, 18, model.SourcePlayer{}.Bot),
				{Name: "", Stats: map[string]int{"frags": 1}},
			},
		},
	}

	rows, err := transformer.BuildRows(context.Background(), match)
	if err != nil {
		t.Fatalf("BuildRows() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}

	row := rows[0]
	if row.NormalizedMode != Mode1v1 {
		t.Fatalf("row.NormalizedMode = %q, want %q", row.NormalizedMode, Mode1v1)
	}
	if row.Frags != 35 || row.Deaths != 12 || row.Kills != 30 || row.TeamKills != 1 || row.Suicides != 2 {
		t.Fatalf("base stats row = %#v", row)
	}
	if row.DamageTaken != 2500 || row.DamageGiven != 3200 || row.SpreeMax != 4 || row.SpreeQuad != 1 {
		t.Fatalf("damage/spree row = %#v", row)
	}
	if row.RLHits != 18 || row.RLKills != 10 || row.LGAttacks != 100 || row.LGHits != 42 {
		t.Fatalf("weapon metrics row = %#v", row)
	}
	if row.GA != 2 || row.RA != 3 || row.YA != 5 || row.Health100 != 6 {
		t.Fatalf("item metrics row = %#v", row)
	}
	if row.Efficiency != 71.42857142857143 {
		t.Fatalf("row.Efficiency = %v, want 71.42857142857143", row.Efficiency)
	}
	if row.LGAccuracy != 42 {
		t.Fatalf("row.LGAccuracy = %v, want 42", row.LGAccuracy)
	}
	if row.ConsolidatedAt.IsZero() || row.ConsolidatedAt.Year() != 2026 {
		t.Fatalf("row.ConsolidatedAt = %v, want injected timestamp", row.ConsolidatedAt)
	}
	if len(row.StatsSnapshot) == 0 {
		t.Fatal("row.StatsSnapshot is empty")
	}
}

func TestBuildRowsMarksBotMatchesWithoutDroppingPlayers(t *testing.T) {
	t.Parallel()

	transformer := NewTransformer()
	match := model.SourceMatch{
		CollectorMatchID: 55,
		HasBots:          false,
		PlayedAt:         time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC),
		Stats: model.SourceStatsMatch{
			Demo: "bot-match.mvd",
			Map:  "dm4",
			Mode: "team",
			DM:   3,
			Players: []model.SourcePlayer{
				testPlayer(t, "BotAlpha", "", "red", map[string]int{"kills": 10}, nil, nil, nil, nil, 0, model.StatsBot{Skill: 7}),
				testPlayer(t, "HumanBravo", "bravo", "blue", map[string]int{"kills": 12}, nil, nil, nil, nil, 0, model.StatsBot{}),
				testPlayer(t, "HumanCharlie", "", "red", map[string]int{"kills": 9}, nil, nil, nil, nil, 0, model.StatsBot{}),
				testPlayer(t, "HumanDelta", "", "blue", map[string]int{"kills": 8}, nil, nil, nil, nil, 0, model.StatsBot{}),
			},
		},
	}

	rows, err := transformer.BuildRows(context.Background(), match)
	if err != nil {
		t.Fatalf("BuildRows() error = %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("len(rows) = %d, want 4", len(rows))
	}
	for _, row := range rows {
		if !row.HasBots || !row.ExcludedFromAnalytics {
			t.Fatalf("row bot flags = %#v, want true/true", row)
		}
	}
}

func TestBuildRowsUsesFixturePayload(t *testing.T) {
	t.Parallel()

	transformer := NewTransformer()
	transformer.now = func() time.Time {
		return time.Date(2026, 3, 26, 16, 0, 0, 0, time.UTC)
	}

	statsMatch := loadFixtureMatchByDemo(t, "2on2_red_vs_fk[dm4]20260319-2038.mvd")
	match := model.SourceMatch{
		CollectorMatchID: 99,
		ServerKey:        "qlash-br-1",
		DemoName:         statsMatch.Demo,
		MapName:          statsMatch.Map,
		RawMode:          "2on2",
		PlayedAt:         time.Date(2026, 3, 19, 23, 48, 44, 0, time.UTC),
		Stats:            statsMatch,
	}

	rows, err := transformer.BuildRows(context.Background(), match)
	if err != nil {
		t.Fatalf("BuildRows() error = %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("len(rows) = %d, want 4", len(rows))
	}

	if rows[0].NormalizedMode != Mode2v2 {
		t.Fatalf("rows[0].NormalizedMode = %q, want %q", rows[0].NormalizedMode, Mode2v2)
	}
	if rows[0].ObservedName != "expert" {
		t.Fatalf("rows[0].ObservedName = %q, want expert", rows[0].ObservedName)
	}
	if rows[0].RLKills != 27 {
		t.Fatalf("rows[0].RLKills = %d, want 27", rows[0].RLKills)
	}
	if rows[1].ObservedName != "MatuzaH" {
		t.Fatalf("rows[1].ObservedName = %q, want MatuzaH", rows[1].ObservedName)
	}
	if rows[1].LGAttacks != 390 || rows[1].LGHits != 117 {
		t.Fatalf("rows[1] LG stats = (%d,%d), want (390,117)", rows[1].LGAttacks, rows[1].LGHits)
	}
}

func testPlayer(
	t *testing.T,
	name string,
	login string,
	team string,
	stats map[string]int,
	damage map[string]int,
	spree map[string]int,
	weapons map[string]any,
	items map[string]any,
	ping int,
	bot model.StatsBot,
) model.SourcePlayer {
	t.Helper()

	return model.SourcePlayer{
		Name:    name,
		Login:   login,
		Team:    team,
		Stats:   stats,
		Damage:  damage,
		Spree:   spree,
		Weapons: mustRawMap(t, weapons),
		Items:   mustRawMap(t, items),
		Ping:    ping,
		Bot:     bot,
	}
}

func mustRawMap(t *testing.T, values map[string]any) map[string]json.RawMessage {
	t.Helper()

	if len(values) == 0 {
		return nil
	}

	result := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("json.Marshal(%s) error = %v", key, err)
		}
		result[key] = data
	}
	return result
}

func loadFixtureMatches(t *testing.T) []model.SourceStatsMatch {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	path := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", "docs", "responses", "laststats.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}

	var matches []model.SourceStatsMatch
	if err := json.Unmarshal(data, &matches); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return matches
}

func loadFixtureMatchByDemo(t *testing.T, demo string) model.SourceStatsMatch {
	t.Helper()

	for _, match := range loadFixtureMatches(t) {
		if match.Demo == demo {
			return match
		}
	}

	t.Fatalf("fixture demo %q not found", demo)
	return model.SourceStatsMatch{}
}
