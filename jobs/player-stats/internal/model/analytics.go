package model

import (
	"encoding/json"
	"time"
)

type Checkpoint struct {
	JobName              string
	LastCollectorMatchID int64
	UpdatedAt            time.Time
}

type Cursor struct {
	LastCollectorMatchID int64
}

type SourceMatch struct {
	CollectorMatchID int64
	ServerKey        string
	DemoName         string
	MapName          string
	RawMode          string
	PlayedAt         time.Time
	HasBots          bool
	StatsPayload     json.RawMessage
	Stats            SourceStatsMatch
	SkipReason       string
}

func (m SourceMatch) Eligible() bool {
	return m.SkipReason == ""
}

type SourceStatsMatch struct {
	Version  int            `json:"version"`
	Date     string         `json:"date"`
	Map      string         `json:"map"`
	Hostname string         `json:"hostname"`
	IP       string         `json:"ip"`
	Port     int            `json:"port"`
	Mode     string         `json:"mode"`
	TL       int            `json:"tl"`
	FL       int            `json:"fl"`
	DM       int            `json:"dm"`
	TP       int            `json:"tp"`
	Duration int            `json:"duration"`
	Demo     string         `json:"demo"`
	Teams    []string       `json:"teams"`
	Players  []SourcePlayer `json:"players"`
}

type SourcePlayer struct {
	Ping    int                        `json:"ping"`
	Login   string                     `json:"login"`
	Name    string                     `json:"name"`
	Team    string                     `json:"team"`
	Stats   map[string]int             `json:"stats"`
	Damage  map[string]int             `json:"dmg"`
	Spree   map[string]int             `json:"spree"`
	Weapons map[string]json.RawMessage `json:"weapons"`
	Items   map[string]json.RawMessage `json:"items"`
	Bot     StatsBot                   `json:"bot"`
}

type StatsBot struct {
	Skill      int  `json:"skill"`
	Customised bool `json:"customised"`
}

type ConsolidationBatch struct {
	Rows       []PlayerMatchRow
	Checkpoint Checkpoint
}

type ResolvePlayerInput struct {
	ObservedName  string
	ObservedLogin string
	ObservedAt    time.Time
}

type PlayerIdentity struct {
	PlayerID    string
	AliasName   string
	Login       string
	Resolution  string
	AliasAction string
}

type PlayerMatchRow struct {
	CollectorMatchID      int64
	ServerKey             string
	DemoName              string
	ObservedName          string
	ObservedLogin         string
	Team                  string
	MapName               string
	RawMode               string
	NormalizedMode        string
	PlayedAt              time.Time
	HasBots               bool
	ExcludedFromAnalytics bool
	Frags                 int
	Deaths                int
	Kills                 int
	TeamKills             int
	Suicides              int
	DamageTaken           int
	DamageGiven           int
	SpreeMax              int
	SpreeQuad             int
	RLHits                int
	RLKills               int
	LGAttacks             int
	LGHits                int
	GA                    int
	RA                    int
	YA                    int
	Health100             int
	Ping                  int
	Efficiency            float64
	LGAccuracy            float64
	StatsSnapshot         json.RawMessage
	ConsolidatedAt        time.Time
}

type PlayerCanonical struct {
	ID           string
	PrimaryLogin string
	DisplayName  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PlayerAlias struct {
	ID          int64
	PlayerID    string
	AliasName   string
	Login       string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
}

type PlayerMatchStat struct {
	ID                    int64
	CollectorMatchID      int64
	PlayerID              string
	ServerKey             string
	DemoName              string
	ObservedName          string
	ObservedLogin         string
	Team                  string
	MapName               string
	RawMode               string
	NormalizedMode        string
	PlayedAt              time.Time
	HasBots               bool
	ExcludedFromAnalytics bool
	Frags                 int
	Deaths                int
	Kills                 int
	TeamKills             int
	Suicides              int
	DamageTaken           int
	DamageGiven           int
	SpreeMax              int
	SpreeQuad             int
	RLHits                int
	RLKills               int
	LGAttacks             int
	LGHits                int
	GA                    int
	RA                    int
	YA                    int
	Health100             int
	Ping                  int
	Efficiency            float64
	LGAccuracy            float64
	StatsSnapshot         json.RawMessage
	ConsolidatedAt        time.Time
}

type BatchResult struct {
	CanonicalInserted int
	CanonicalReused   int
	AliasesInserted   int
	AliasesUpdated    int
	StatsInserted     int
	StatsUpdated      int
}
