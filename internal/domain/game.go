package domain

import "time"

// GameState is the lifecycle state of a game.
type GameState string

// Known game states.
const (
	GameStateScheduled GameState = "scheduled"
	GameStateLive      GameState = "live"
	GameStateFinal     GameState = "final"
	GameStatePostponed GameState = "postponed"
	GameStateCanceled  GameState = "canceled"
)

// Side identifies which side of a matchup a team is on. "neutral"
// means a neutral-site game — the Home field still carries the home
// side as the data source labeled it, but Side is Neutral for both.
type Side string

// Known sides.
const (
	SideHome    Side = "home"
	SideAway    Side = "away"
	SideNeutral Side = "neutral"
)

// Team is one side of a game.
type Team struct {
	// Seo is NCAA.com's URL slug for the team ("notre-dame",
	// "utah-state"). It's the primary key for team-centric lookups.
	Seo string `json:"seo"`

	// Short is a display name ("Notre Dame").
	Short string `json:"short"`

	// Rank is the AP/Coaches rank at kickoff, "" if unranked. Preserved
	// as a string because the source sometimes carries "T" prefixes for
	// ties ("T19").
	Rank string `json:"rank,omitempty"`

	// Score is the team's final/running score. Zero before the game
	// starts; use the enclosing Game.State to disambiguate from a
	// 0-point result.
	Score int `json:"score"`

	// Winner is true when this team won. Both false in a tie or when
	// the game isn't final.
	Winner bool `json:"winner,omitempty"`

	// Conference is the team's conference slug ("acc", "sec"). Empty
	// when the source didn't carry it.
	Conference string `json:"conference,omitempty"`
}

// Game is a single scheduled or completed game. Sport-agnostic — every
// sport normalizer produces this shape.
type Game struct {
	// ID is NCAA.com's game identifier. Stable across refetches.
	ID string `json:"id"`

	// Date is the game date in the venue's local timezone, truncated
	// to midnight so day-level grouping is stable regardless of
	// kickoff time.
	Date time.Time `json:"date"`

	// StartTime is the kickoff instant in UTC. Zero if unknown or TBA.
	StartTime time.Time `json:"start_time,omitempty"`

	// State is the lifecycle state.
	State GameState `json:"state"`

	// Home and Away are the two sides. For neutral-site games the
	// labels still match the data source's assignment; the Neutral
	// field below carries the neutral-site flag.
	Home Team `json:"home"`
	Away Team `json:"away"`

	// Neutral is true for neutral-site games (conference tournaments,
	// NCAA tournament).
	Neutral bool `json:"neutral,omitempty"`

	// IsExhibition is true for exhibition / preseason / intra-squad
	// games. The API filters these out by default via the
	// `exclude_exhibitions=true` query parameter.
	IsExhibition bool `json:"is_exhibition,omitempty"`
}
