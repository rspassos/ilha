package merge

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rspassos/ilha/jobs/collector/internal/config"
	"github.com/rspassos/ilha/jobs/collector/internal/model"
)

const (
	WarningScoreOnly        = "score_only"
	WarningStatsOnly        = "stats_only"
	WarningDuplicateScores  = "duplicate_scores"
	WarningDuplicateStats   = "duplicate_stats"
	WarningModeMismatch     = "mode_mismatch"
	WarningMapMismatch      = "map_mismatch"
	WarningPlayedAtMismatch = "played_at_mismatch"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Merge(server config.ServerConfig, scores []model.ScoreMatch, stats []model.StatsMatch) ([]model.MatchRecord, []model.MergeWarning, error) {
	scoreIndex, scoreWarnings := indexScores(server, scores)
	statsIndex, statsWarnings := indexStats(server, stats)

	demos := make(map[string]struct{}, len(scoreIndex)+len(statsIndex))
	for demo := range scoreIndex {
		demos[demo] = struct{}{}
	}
	for demo := range statsIndex {
		demos[demo] = struct{}{}
	}

	orderedDemos := make([]string, 0, len(demos))
	for demo := range demos {
		orderedDemos = append(orderedDemos, demo)
	}
	sort.Strings(orderedDemos)

	warnings := append(scoreWarnings, statsWarnings...)
	records := make([]model.MatchRecord, 0, len(orderedDemos))
	for _, demo := range orderedDemos {
		record, recordWarnings, err := buildRecord(server, demo, scoreIndex[demo], statsIndex[demo])
		if err != nil {
			return nil, nil, fmt.Errorf("merge demo %q for server %q: %w", demo, server.Key, err)
		}
		records = append(records, record)
		warnings = append(warnings, recordWarnings...)
	}

	return records, warnings, nil
}

func indexScores(server config.ServerConfig, scores []model.ScoreMatch) (map[string]model.ScoreMatch, []model.MergeWarning) {
	index := make(map[string]model.ScoreMatch, len(scores))
	warnings := make([]model.MergeWarning, 0)
	for _, score := range scores {
		if _, exists := index[score.Demo]; exists {
			warnings = append(warnings, model.MergeWarning{
				ServerKey: server.Key,
				DemoName:  score.Demo,
				Reason:    WarningDuplicateScores,
			})
		}
		index[score.Demo] = score
	}
	return index, warnings
}

func indexStats(server config.ServerConfig, stats []model.StatsMatch) (map[string]model.StatsMatch, []model.MergeWarning) {
	index := make(map[string]model.StatsMatch, len(stats))
	warnings := make([]model.MergeWarning, 0)
	for _, stat := range stats {
		if _, exists := index[stat.Demo]; exists {
			warnings = append(warnings, model.MergeWarning{
				ServerKey: server.Key,
				DemoName:  stat.Demo,
				Reason:    WarningDuplicateStats,
			})
		}
		index[stat.Demo] = stat
	}
	return index, warnings
}

func buildRecord(server config.ServerConfig, demo string, score model.ScoreMatch, stats model.StatsMatch) (model.MatchRecord, []model.MergeWarning, error) {
	recordWarnings := make([]model.MergeWarning, 0, 4)

	hasScore := strings.TrimSpace(score.Demo) != ""
	hasStats := strings.TrimSpace(stats.Demo) != ""

	switch {
	case hasScore && !hasStats:
		recordWarnings = append(recordWarnings, warning(server.Key, demo, WarningScoreOnly))
	case hasStats && !hasScore:
		recordWarnings = append(recordWarnings, warning(server.Key, demo, WarningStatsOnly))
	}

	if hasScore && hasStats {
		if score.Mode != "" && stats.Mode != "" && !strings.EqualFold(score.Mode, stats.Mode) {
			recordWarnings = append(recordWarnings, warning(server.Key, demo, WarningModeMismatch))
		}
		if score.Map != "" && stats.Map != "" && !strings.EqualFold(score.Map, stats.Map) {
			recordWarnings = append(recordWarnings, warning(server.Key, demo, WarningMapMismatch))
		}
		if !score.PlayedAt.IsZero() && !stats.PlayedAt.IsZero() && !score.PlayedAt.Equal(stats.PlayedAt) {
			recordWarnings = append(recordWarnings, warning(server.Key, demo, WarningPlayedAtMismatch))
		}
	}

	scorePayload, err := marshalPayload(score, hasScore)
	if err != nil {
		return model.MatchRecord{}, nil, fmt.Errorf("marshal score payload: %w", err)
	}
	statsPayload, err := marshalPayload(stats, hasStats)
	if err != nil {
		return model.MatchRecord{}, nil, fmt.Errorf("marshal stats payload: %w", err)
	}

	record := model.MatchRecord{
		ServerKey:       server.Key,
		ServerName:      server.Name,
		DemoName:        demo,
		MatchKey:        buildMatchKey(server.Key, demo),
		Mode:            deriveMode(score, stats),
		MapName:         deriveMap(score, stats),
		Participants:    deriveParticipants(score, stats),
		PlayedAt:        derivePlayedAt(score, stats),
		DurationSeconds: deriveDuration(stats),
		Hostname:        strings.TrimSpace(stats.Hostname),
		HasBots:         hasBots(score),
		ScorePayload:    scorePayload,
		StatsPayload:    statsPayload,
	}

	record.MergedPayload, err = marshalMergedPayload(record, hasScore, hasStats)
	if err != nil {
		return model.MatchRecord{}, nil, fmt.Errorf("marshal merged payload: %w", err)
	}

	return record, recordWarnings, nil
}

func marshalPayload[T any](payload T, ok bool) (json.RawMessage, error) {
	if !ok {
		return nil, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func marshalMergedPayload(record model.MatchRecord, hasScore bool, hasStats bool) (json.RawMessage, error) {
	payload := struct {
		ServerKey       string          `json:"server_key"`
		ServerName      string          `json:"server_name"`
		DemoName        string          `json:"demo_name"`
		MatchKey        string          `json:"match_key"`
		Mode            string          `json:"mode"`
		MapName         string          `json:"map_name"`
		Participants    string          `json:"participants"`
		PlayedAt        string          `json:"played_at"`
		DurationSeconds int             `json:"duration_seconds"`
		Hostname        string          `json:"hostname"`
		HasBots         bool            `json:"has_bots"`
		HasScore        bool            `json:"has_score"`
		HasStats        bool            `json:"has_stats"`
		ScorePayload    json.RawMessage `json:"score_payload,omitempty"`
		StatsPayload    json.RawMessage `json:"stats_payload,omitempty"`
	}{
		ServerKey:       record.ServerKey,
		ServerName:      record.ServerName,
		DemoName:        record.DemoName,
		MatchKey:        record.MatchKey,
		Mode:            record.Mode,
		MapName:         record.MapName,
		Participants:    record.Participants,
		PlayedAt:        record.PlayedAt.UTC().Format("2006-01-02T15:04:05Z"),
		DurationSeconds: record.DurationSeconds,
		Hostname:        record.Hostname,
		HasBots:         record.HasBots,
		HasScore:        hasScore,
		HasStats:        hasStats,
		ScorePayload:    record.ScorePayload,
		StatsPayload:    record.StatsPayload,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func buildMatchKey(serverKey string, demo string) string {
	return strings.TrimSpace(serverKey) + ":" + strings.TrimSpace(demo)
}

func deriveMode(score model.ScoreMatch, stats model.StatsMatch) string {
	if value := strings.TrimSpace(score.Mode); value != "" {
		return value
	}
	return strings.TrimSpace(stats.Mode)
}

func deriveMap(score model.ScoreMatch, stats model.StatsMatch) string {
	if value := strings.TrimSpace(score.Map); value != "" {
		return value
	}
	return strings.TrimSpace(stats.Map)
}

func deriveParticipants(score model.ScoreMatch, stats model.StatsMatch) string {
	if value := strings.TrimSpace(score.Participants); value != "" {
		return value
	}
	if len(stats.Teams) > 0 {
		return strings.Join(stats.Teams, " vs ")
	}
	names := make([]string, 0, len(stats.Players))
	for _, player := range stats.Players {
		name := strings.TrimSpace(player.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, " vs ")
}

func derivePlayedAt(score model.ScoreMatch, stats model.StatsMatch) time.Time {
	if !score.PlayedAt.IsZero() {
		return score.PlayedAt
	}
	return stats.PlayedAt
}

func deriveDuration(stats model.StatsMatch) int {
	return stats.Duration
}

func hasBots(score model.ScoreMatch) bool {
	for _, player := range score.Players {
		if player.IsBot {
			return true
		}
	}
	for _, team := range score.Teams {
		for _, player := range team.Players {
			if player.IsBot {
				return true
			}
		}
	}
	return false
}

func warning(serverKey string, demo string, reason string) model.MergeWarning {
	return model.MergeWarning{
		ServerKey: serverKey,
		DemoName:  demo,
		Reason:    reason,
	}
}
