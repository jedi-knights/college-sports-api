// Package ports declares the interfaces inbound and outbound adapters
// implement. The interfaces live here rather than next to their
// implementations so the direction of dependency points inward
// (adapters → ports → domain), per hexagonal architecture.
package ports

import (
	"context"
	"time"

	"github.com/jedi-knights/college-sports-api/internal/domain"
)

// ScoreboardFetcher returns every game played on a given date for one
// sport-division pair. Implementations may hit NCAA.com directly, a
// third-party proxy, or a cache in front of either.
//
// Date is interpreted as the game date in the venue's local timezone,
// truncated to midnight. Day-boundary games (west-coast late starts)
// are reported under the date NCAA.com assigns them.
//
// Returns an empty slice and no error when there are no games that
// day. Implementations should return a wrapped apperrors.AppError
// (BAD_REQUEST on invalid input, UNAVAILABLE on upstream failure,
// NOT_FOUND when the sport/division pair is unknown).
type ScoreboardFetcher interface {
	FetchDaily(ctx context.Context, sport domain.SportID, div domain.Division, date time.Time) ([]domain.Game, error)
}

// TeamScheduleFetcher returns every game in a season for one team.
// The seo slug is NCAA.com's URL-safe team identifier.
//
// Implementations may derive the schedule from a team-filtered
// upstream query or by composing day-by-day scoreboards within the
// Sport's SeasonBoundaries.
type TeamScheduleFetcher interface {
	FetchForTeam(ctx context.Context, sport domain.SportID, div domain.Division, seo string, year int) ([]domain.Game, error)
}
