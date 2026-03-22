package model

import (
	"encoding/json"
	"time"
)

type MatchRecord struct {
	ServerKey       string          `json:"server_key"`
	ServerName      string          `json:"server_name"`
	DemoName        string          `json:"demo_name"`
	MatchKey        string          `json:"match_key"`
	Mode            string          `json:"mode"`
	MapName         string          `json:"map_name"`
	Participants    string          `json:"participants"`
	PlayedAt        time.Time       `json:"played_at"`
	DurationSeconds int             `json:"duration_seconds"`
	Hostname        string          `json:"hostname"`
	HasBots         bool            `json:"has_bots"`
	ScorePayload    json.RawMessage `json:"score_payload"`
	StatsPayload    json.RawMessage `json:"stats_payload"`
	MergedPayload   json.RawMessage `json:"merged_payload"`
}

type MergeWarning struct {
	ServerKey string `json:"server_key"`
	DemoName  string `json:"demo_name"`
	Reason    string `json:"reason"`
}
