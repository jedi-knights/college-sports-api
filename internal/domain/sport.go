// Package domain holds sport-agnostic value types shared across every
// adapter. Nothing in here imports an adapter package — this is the
// center of the hexagon.
package domain

import "time"

// SportID is the canonical identifier for a sport-and-side pair:
// "soccer-men", "soccer-women", "basketball-men", etc. The hyphenated
// form matches the URL path segment used in the public API so a
// scoreboard URL like /v1/scoreboard/soccer-women/d1/... never needs
// a lookup table to resolve the sport.
type SportID string

// Known sport IDs. The list grows as new sport packages register.
const (
	SportSoccerMen   SportID = "soccer-men"
	SportSoccerWomen SportID = "soccer-women"
)

// Division is the NCAA competitive division: "d1", "d2", or "d3".
type Division string

// Known divisions.
const (
	DivisionD1 Division = "d1"
	DivisionD2 Division = "d2"
	DivisionD3 Division = "d3"
)

// SeasonWindow is the inclusive date range of a sport's regular + post
// season for a given year. Used to bound day-by-day scoreboard crawls
// so we don't fetch dates no games are played on.
type SeasonWindow struct {
	Start time.Time
	End   time.Time
}

// Sport is the pluggability seam. Each sport package exports an
// implementation and registers it with the container at startup. The
// interface stays thin on purpose — capability-shaped behaviors
// (scoreboard fetching, standings computation, RPI) live on separate
// typed maps in the container, not on this interface.
type Sport interface {
	// ID returns the sport's canonical identifier.
	ID() SportID

	// Divisions returns the NCAA divisions this sport covers.
	Divisions() []Division

	// SeasonBoundaries returns the regular-plus-post-season window for
	// the given season year.
	SeasonBoundaries(year int) SeasonWindow
}
