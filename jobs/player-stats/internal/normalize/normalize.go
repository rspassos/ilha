package normalize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
)

const (
	Mode1v1  = "1on1"
	Mode2v2  = "2on2"
	Mode3v3  = "3on3"
	Mode4v4  = "4on4"
	ModeDMM4 = "dmm4"
)

type Transformer struct {
	now func() time.Time
}

func NewTransformer() *Transformer {
	return &Transformer{
		now: time.Now().UTC,
	}
}

func (t *Transformer) BuildRows(_ context.Context, match model.SourceMatch) ([]model.PlayerMatchRow, error) {
	if !match.Eligible() {
		return nil, fmt.Errorf("match %d is not eligible for consolidation", match.CollectorMatchID)
	}

	players := activePlayers(match.Stats.Players)
	if len(players) == 0 {
		return nil, fmt.Errorf("match %d has no active players", match.CollectorMatchID)
	}

	normalizedMode, err := NormalizeMode(match.Stats.DM, len(players))
	if err != nil {
		return nil, fmt.Errorf("normalize mode for match %d: %w", match.CollectorMatchID, err)
	}

	matchHasBots := match.HasBots || hasBotPlayers(players)
	consolidatedAt := time.Time{}
	if t != nil && t.now != nil {
		consolidatedAt = t.now()
	}
	if consolidatedAt.IsZero() {
		consolidatedAt = time.Now().UTC()
	}

	rows := make([]model.PlayerMatchRow, 0, len(players))
	for _, player := range players {
		snapshot, err := json.Marshal(player)
		if err != nil {
			return nil, fmt.Errorf("marshal player snapshot for match %d player %q: %w", match.CollectorMatchID, player.Name, err)
		}

		row := model.PlayerMatchRow{
			CollectorMatchID:      match.CollectorMatchID,
			ServerKey:             strings.TrimSpace(match.ServerKey),
			DemoName:              firstNonEmpty(match.DemoName, match.Stats.Demo),
			ObservedName:          strings.TrimSpace(player.Name),
			ObservedLogin:         strings.TrimSpace(player.Login),
			Team:                  strings.TrimSpace(player.Team),
			MapName:               firstNonEmpty(match.MapName, match.Stats.Map),
			RawMode:               firstNonEmpty(match.RawMode, match.Stats.Mode),
			NormalizedMode:        normalizedMode,
			PlayedAt:              match.PlayedAt,
			HasBots:               matchHasBots,
			ExcludedFromAnalytics: matchHasBots,
			Frags:                 statValue(player.Stats, "frags"),
			Deaths:                statValue(player.Stats, "deaths"),
			Kills:                 statValue(player.Stats, "kills"),
			TeamKills:             statValue(player.Stats, "tk"),
			Suicides:              statValue(player.Stats, "suicides"),
			DamageTaken:           statValue(player.Damage, "taken"),
			DamageGiven:           statValue(player.Damage, "given"),
			SpreeMax:              statValue(player.Spree, "max"),
			SpreeQuad:             statValue(player.Spree, "quad"),
			RLHits:                nestedValue(player.Weapons["rl"], "acc", "hits"),
			RLKills:               nestedValue(player.Weapons["rl"], "kills", "total"),
			LGAttacks:             nestedValue(player.Weapons["lg"], "acc", "attacks"),
			LGHits:                nestedValue(player.Weapons["lg"], "acc", "hits"),
			GA:                    nestedValue(player.Items["ga"], "took"),
			RA:                    nestedValue(player.Items["ra"], "took"),
			YA:                    nestedValue(player.Items["ya"], "took"),
			Health100:             nestedValue(player.Items["health_100"], "took"),
			Ping:                  player.Ping,
			StatsSnapshot:         snapshot,
			ConsolidatedAt:        consolidatedAt,
		}
		row.Efficiency = ratioPercentage(row.Kills, row.Kills+row.Deaths)
		row.LGAccuracy = ratioPercentage(row.LGHits, row.LGAttacks)
		rows = append(rows, row)
	}

	return rows, nil
}

func NormalizeMode(dm int, activePlayers int) (string, error) {
	if dm == 4 {
		return ModeDMM4, nil
	}

	switch activePlayers {
	case 2:
		return Mode1v1, nil
	case 4:
		return Mode2v2, nil
	case 6:
		return Mode3v3, nil
	case 8:
		return Mode4v4, nil
	default:
		return "", fmt.Errorf("unsupported active player count %d", activePlayers)
	}
}

func activePlayers(players []model.SourcePlayer) []model.SourcePlayer {
	active := make([]model.SourcePlayer, 0, len(players))
	for _, player := range players {
		if !isActivePlayer(player) {
			continue
		}
		active = append(active, player)
	}
	return active
}

func isActivePlayer(player model.SourcePlayer) bool {
	if strings.TrimSpace(player.Name) == "" {
		return false
	}
	return len(player.Stats) > 0
}

func hasBotPlayers(players []model.SourcePlayer) bool {
	for _, player := range players {
		if player.Bot.Skill > 0 || player.Bot.Customised {
			return true
		}
	}
	return false
}

func statValue(values map[string]int, key string) int {
	if len(values) == 0 {
		return 0
	}
	return values[key]
}

func nestedValue(payload json.RawMessage, path ...string) int {
	if len(payload) == 0 || len(path) == 0 {
		return 0
	}

	var current any
	if err := json.Unmarshal(payload, &current); err != nil {
		return 0
	}

	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return 0
		}
		current, ok = object[key]
		if !ok {
			return 0
		}
	}

	return numberToInt(current)
}

func numberToInt(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	default:
		return 0
	}
}

func ratioPercentage(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) * 100 / float64(denominator)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var ErrTransformerNotInitialized = errors.New("transformer is not initialized")

func (t *Transformer) Validate() error {
	if t == nil {
		return ErrTransformerNotInitialized
	}
	if t.now == nil {
		return ErrTransformerNotInitialized
	}
	return nil
}
