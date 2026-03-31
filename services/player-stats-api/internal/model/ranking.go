package model

import "time"

const (
	DefaultSortBy        = "efficiency"
	DefaultSortDirection = "desc"
)

type RankingQuery struct {
	Mode           string
	Map            string
	Server         string
	From           time.Time
	To             time.Time
	SortBy         string
	SortDirection  string
	Limit          int
	Offset         int
	MinimumMatches int
}

type PlayerRankingRow struct {
	PlayerID    string  `json:"player_id"`
	DisplayName string  `json:"display_name"`
	Matches     int     `json:"matches"`
	Efficiency  float64 `json:"efficiency"`
	Frags       int     `json:"frags"`
	Kills       int     `json:"kills"`
	Deaths      int     `json:"deaths"`
	LGAccuracy  float64 `json:"lg_accuracy"`
	RLHits      int     `json:"rl_hits"`
	Rank        int     `json:"rank"`
}

type RankingPage struct {
	Data    []PlayerRankingRow `json:"data"`
	Meta    PaginationMeta     `json:"meta"`
	Filters AppliedFilters     `json:"filters"`
}

type PaginationMeta struct {
	SortBy         string `json:"sort_by"`
	SortDirection  string `json:"sort_direction"`
	MinimumMatches int    `json:"minimum_matches"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
	Returned       int    `json:"returned"`
	HasNext        bool   `json:"has_next"`
}

type AppliedFilters struct {
	Mode   string     `json:"mode,omitempty"`
	Map    string     `json:"map,omitempty"`
	Server string     `json:"server,omitempty"`
	From   *time.Time `json:"from,omitempty"`
	To     *time.Time `json:"to,omitempty"`
}

func NewRankingPage(query RankingQuery, rows []PlayerRankingRow, hasNext bool) RankingPage {
	return RankingPage{
		Data: rows,
		Meta: PaginationMeta{
			SortBy:         query.SortBy,
			SortDirection:  query.SortDirection,
			MinimumMatches: query.MinimumMatches,
			Limit:          query.Limit,
			Offset:         query.Offset,
			Returned:       len(rows),
			HasNext:        hasNext,
		},
		Filters: NewAppliedFilters(query),
	}
}

func NewAppliedFilters(query RankingQuery) AppliedFilters {
	filters := AppliedFilters{
		Mode:   query.Mode,
		Map:    query.Map,
		Server: query.Server,
	}
	if !query.From.IsZero() {
		from := query.From.UTC()
		filters.From = &from
	}
	if !query.To.IsZero() {
		to := query.To.UTC()
		filters.To = &to
	}
	return filters
}
