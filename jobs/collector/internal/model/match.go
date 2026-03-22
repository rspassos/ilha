package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const statsDateLayout = "2006-01-02 15:04:05 -0700"

type ScoreMatch struct {
	Title        string        `json:"title"`
	Demo         string        `json:"demo"`
	Timestamp    string        `json:"timestamp"`
	TimestampISO string        `json:"timestamp_iso"`
	PlayedAt     time.Time     `json:"-"`
	Mode         string        `json:"mode"`
	Participants string        `json:"participants"`
	Map          string        `json:"map"`
	Scores       string        `json:"scores"`
	Players      []ScorePlayer `json:"players"`
	Teams        []ScoreTeam   `json:"teams"`
}

type ScorePlayer struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	NameColor string `json:"name_color"`
	Team      string `json:"team"`
	TeamColor string `json:"team_color"`
	Skin      string `json:"skin"`
	Colors    []int  `json:"colors"`
	Frags     int    `json:"frags"`
	Ping      int    `json:"ping"`
	Time      int    `json:"time"`
	CC        string `json:"cc"`
	IsBot     bool   `json:"is_bot"`
}

type ScoreTeam struct {
	Name      string        `json:"name"`
	NameColor string        `json:"name_color"`
	Frags     int           `json:"frags"`
	Ping      int           `json:"ping"`
	Colors    []int         `json:"colors"`
	Players   []ScorePlayer `json:"players"`
}

type StatsMatch struct {
	Version  int           `json:"version"`
	Date     string        `json:"date"`
	PlayedAt time.Time     `json:"-"`
	Map      string        `json:"map"`
	Hostname string        `json:"hostname"`
	IP       string        `json:"ip"`
	Port     int           `json:"port"`
	Mode     string        `json:"mode"`
	TL       int           `json:"tl"`
	FL       int           `json:"fl"`
	DM       int           `json:"dm"`
	TP       int           `json:"tp"`
	Duration int           `json:"duration"`
	Demo     string        `json:"demo"`
	Teams    []string      `json:"teams"`
	Players  []StatsPlayer `json:"players"`
}

type StatsPlayer struct {
	TopColor    int                        `json:"top-color"`
	BottomColor int                        `json:"bottom-color"`
	Ping        int                        `json:"ping"`
	Login       string                     `json:"login"`
	Name        string                     `json:"name"`
	Team        string                     `json:"team"`
	Stats       map[string]int             `json:"stats"`
	Damage      map[string]int             `json:"dmg"`
	XferRL      int                        `json:"xferRL"`
	XferLG      int                        `json:"xferLG"`
	Spree       map[string]int             `json:"spree"`
	Control     float64                    `json:"control"`
	Speed       StatsSpeed                 `json:"speed"`
	Weapons     map[string]json.RawMessage `json:"weapons"`
	Items       map[string]json.RawMessage `json:"items"`
	Bot         StatsBot                   `json:"bot"`
}

type StatsSpeed struct {
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

type StatsBot struct {
	Skill      int  `json:"skill"`
	Customised bool `json:"customised"`
}

func (m *ScoreMatch) Normalize() error {
	if m == nil {
		return errors.New("score match is nil")
	}
	m.Title = strings.TrimSpace(m.Title)
	m.Demo = strings.TrimSpace(m.Demo)
	m.Timestamp = strings.TrimSpace(m.Timestamp)
	m.TimestampISO = strings.TrimSpace(m.TimestampISO)
	m.Mode = strings.TrimSpace(m.Mode)
	m.Participants = strings.TrimSpace(m.Participants)
	m.Map = strings.TrimSpace(m.Map)
	m.Scores = strings.TrimSpace(m.Scores)

	if m.Demo == "" {
		return errors.New("demo must not be empty")
	}
	if m.TimestampISO == "" {
		return fmt.Errorf("score match %q: timestamp_iso must not be empty", m.Demo)
	}

	playedAt, err := time.Parse(time.RFC3339, m.TimestampISO)
	if err != nil {
		return fmt.Errorf("score match %q: parse timestamp_iso: %w", m.Demo, err)
	}
	m.PlayedAt = playedAt

	return nil
}

func (m *StatsMatch) Normalize() error {
	if m == nil {
		return errors.New("stats match is nil")
	}
	m.Date = strings.TrimSpace(m.Date)
	m.Map = strings.TrimSpace(m.Map)
	m.Hostname = strings.TrimSpace(m.Hostname)
	m.IP = strings.TrimSpace(m.IP)
	m.Mode = strings.TrimSpace(m.Mode)
	m.Demo = strings.TrimSpace(m.Demo)

	if m.Demo == "" {
		return errors.New("demo must not be empty")
	}
	if m.Date == "" {
		return fmt.Errorf("stats match %q: date must not be empty", m.Demo)
	}

	playedAt, err := time.Parse(statsDateLayout, m.Date)
	if err != nil {
		return fmt.Errorf("stats match %q: parse date: %w", m.Demo, err)
	}
	m.PlayedAt = playedAt

	return nil
}
