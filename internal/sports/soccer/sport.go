// Package soccer wires men's and women's college soccer into the
// container's capability maps. The sport-specific logic lives here —
// season boundaries, division list, future home for standings and
// RPI (Phase 4). Keep adapters sport-agnostic so this file is the
// only place that encodes soccer rules.
package soccer

import (
	"time"

	"github.com/jedi-knights/college-sports-api/internal/domain"
)

// Men is the sport descriptor for NCAA men's college soccer.
type Men struct{}

// ID returns the canonical identifier.
func (Men) ID() domain.SportID { return domain.SportSoccerMen }

// Divisions returns the divisions this sport covers.
func (Men) Divisions() []domain.Division {
	return []domain.Division{domain.DivisionD1, domain.DivisionD2, domain.DivisionD3}
}

// SeasonBoundaries returns the regular + post-season window for
// year. NCAA men's soccer runs late August through the College Cup in
// mid-December.
func (Men) SeasonBoundaries(year int) domain.SeasonWindow {
	return seasonWindow(year)
}

// Women is the sport descriptor for NCAA women's college soccer.
type Women struct{}

// ID returns the canonical identifier.
func (Women) ID() domain.SportID { return domain.SportSoccerWomen }

// Divisions returns the divisions this sport covers.
func (Women) Divisions() []domain.Division {
	return []domain.Division{domain.DivisionD1, domain.DivisionD2, domain.DivisionD3}
}

// SeasonBoundaries returns the regular + post-season window for
// year. Women's soccer runs on the same calendar as men's.
func (Women) SeasonBoundaries(year int) domain.SeasonWindow {
	return seasonWindow(year)
}

// seasonWindow is the shared men's/women's soccer season window.
// Both sides play from early August through mid-December in the same
// calendar year.
func seasonWindow(year int) domain.SeasonWindow {
	return domain.SeasonWindow{
		Start: time.Date(year, time.August, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(year, time.December, 20, 0, 0, 0, 0, time.UTC),
	}
}
