// Package container is the DI composition root. It holds the typed
// capability maps that keep adding a new sport to a one-call-site
// change, without a god Sport interface.
package container

import (
	"github.com/jedi-knights/college-sports-api/internal/domain"
	"github.com/jedi-knights/college-sports-api/internal/ports"
)

// Registry owns the sport descriptors and their capabilities. Each
// capability map is optional: a sport without a StandingsComputer
// just doesn't appear in that map, and the handler layer returns 404
// for the corresponding endpoint. No fake nil-returning
// implementations.
type Registry struct {
	sports               map[domain.SportID]domain.Sport
	scoreboardFetchers   map[domain.SportID]ports.ScoreboardFetcher
	teamScheduleFetchers map[domain.SportID]ports.TeamScheduleFetcher
}

// NewRegistry returns an empty Registry. Call RegisterSport and the
// Register* capability methods during startup; the registry is
// treated as read-only after the container is built.
func NewRegistry() *Registry {
	return &Registry{
		sports:               make(map[domain.SportID]domain.Sport),
		scoreboardFetchers:   make(map[domain.SportID]ports.ScoreboardFetcher),
		teamScheduleFetchers: make(map[domain.SportID]ports.TeamScheduleFetcher),
	}
}

// RegisterSport records a sport descriptor. Idempotent: later calls
// overwrite earlier ones for the same ID. Returns the registry for
// fluent chaining in `main`.
func (r *Registry) RegisterSport(s domain.Sport) *Registry {
	r.sports[s.ID()] = s
	return r
}

// RegisterScoreboardFetcher attaches a scoreboard fetcher to the
// given sport. The sport must already be registered via
// RegisterSport.
func (r *Registry) RegisterScoreboardFetcher(id domain.SportID, f ports.ScoreboardFetcher) *Registry {
	r.scoreboardFetchers[id] = f
	return r
}

// RegisterTeamScheduleFetcher attaches a team-schedule fetcher.
func (r *Registry) RegisterTeamScheduleFetcher(id domain.SportID, f ports.TeamScheduleFetcher) *Registry {
	r.teamScheduleFetchers[id] = f
	return r
}

// Sport returns the sport descriptor for id, if registered.
func (r *Registry) Sport(id domain.SportID) (domain.Sport, bool) {
	s, ok := r.sports[id]
	return s, ok
}

// ScoreboardFetcher returns the scoreboard fetcher for id, if any.
// Absence is how we signal "this sport doesn't support scoreboard
// reads" — the handler translates it to 404.
func (r *Registry) ScoreboardFetcher(id domain.SportID) (ports.ScoreboardFetcher, bool) {
	f, ok := r.scoreboardFetchers[id]
	return f, ok
}

// TeamScheduleFetcher returns the team-schedule fetcher for id, if any.
func (r *Registry) TeamScheduleFetcher(id domain.SportID) (ports.TeamScheduleFetcher, bool) {
	f, ok := r.teamScheduleFetchers[id]
	return f, ok
}

// SportIDs returns every registered sport ID. Order is not stable.
func (r *Registry) SportIDs() []domain.SportID {
	out := make([]domain.SportID, 0, len(r.sports))
	for id := range r.sports {
		out = append(out, id)
	}
	return out
}
