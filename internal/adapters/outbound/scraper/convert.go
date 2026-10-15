package scraper

import (
	"strconv"
	"strings"
	"time"

	"github.com/jedi-knights/college-sports-api/internal/domain"
)

// convertGame turns an NCAA proxy game into a sport-agnostic
// domain.Game. The second return is false when the game is missing
// fields we require (ID, both team seo slugs) — such rows are skipped
// rather than emitted as half-populated records.
//
// Date is the request date; used as the game's local-day grouping
// value when the upstream doesn't carry a parseable startDate.
func convertGame(src ncaaGame, requestDate time.Time) (domain.Game, bool) {
	if src.GameID == "" {
		return domain.Game{}, false
	}
	home := convertTeam(src.Home)
	away := convertTeam(src.Away)
	if home.Seo == "" && away.Seo == "" {
		// Both sides missing a slug — not useful downstream.
		return domain.Game{}, false
	}

	g := domain.Game{
		ID:        src.GameID,
		Date:      parseGameDate(src.StartDate, requestDate),
		StartTime: parseStartTime(src.StartTimeEpoch),
		State:     convertState(src.GameState),
		Home:      home,
		Away:      away,
	}
	return g, true
}

func convertTeam(src ncaaTeam) domain.Team {
	t := domain.Team{
		Seo:    src.Names.Seo,
		Short:  pickShort(src.Names.Short, src.Names.Full, src.Names.Char6),
		Rank:   src.Rank,
		Winner: src.Winner,
	}
	if s := strings.TrimSpace(src.Score); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			t.Score = n
		}
	}
	if len(src.Conferences) > 0 {
		t.Conference = src.Conferences[0].ConferenceSeo
	}
	return t
}

func pickShort(parts ...string) string {
	for _, s := range parts {
		if s = strings.TrimSpace(s); s != "" {
			return s
		}
	}
	return ""
}

// convertState maps NCAA's string state to our GameState. Unknown
// strings fall through to Scheduled rather than erroring.
func convertState(raw string) domain.GameState {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "final":
		return domain.GameStateFinal
	case "live", "in_progress", "in-progress":
		return domain.GameStateLive
	case "postponed":
		return domain.GameStatePostponed
	case "canceled":
		return domain.GameStateCanceled
	default:
		return domain.GameStateScheduled
	}
}

// parseGameDate interprets NCAA's "MM/DD/YYYY" startDate. Falls back
// to the request date when the string is missing or malformed.
func parseGameDate(startDate string, fallback time.Time) time.Time {
	s := strings.TrimSpace(startDate)
	if s == "" {
		return truncateToDay(fallback)
	}
	t, err := time.Parse("01/02/2006", s)
	if err != nil {
		return truncateToDay(fallback)
	}
	return t
}

func parseStartTime(epochStr string) time.Time {
	s := strings.TrimSpace(epochStr)
	if s == "" {
		return time.Time{}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n == 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
