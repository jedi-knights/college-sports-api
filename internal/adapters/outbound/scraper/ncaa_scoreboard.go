// Package scraper contains ScoreboardFetcher implementations that pull
// game data from NCAA.com. The architecture target is a Composite that
// tries NCAA's GraphQL endpoint first, falls back to the legacy REST
// API, and finally parses rendered HTML — matching the Phase 2 plan.
//
// Phase 1 ships a single implementation (NCAAScoreboard) that hits the
// community-maintained ncaa-api.henrygd.me proxy. It returns the same
// data NCAA.com's own front end consumes and lets us focus Phase 1 on
// the server shape (caching, handlers, deployment) rather than
// reverse-engineering NCAA's persisted-query hashes.
//
// Phase 2 adds `graphql.go` (direct NCAA access) and composes it in
// front of this one; the interface stays the same.
package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jedi-knights/go-platform/apperrors"

	"github.com/jedi-knights/college-sports-api/internal/domain"
)

// DefaultNCAABaseURL is the community proxy we use as the Phase-1 data
// source. Overridable at construction so tests can inject a stub HTTP
// server.
const DefaultNCAABaseURL = "https://ncaa-api.henrygd.me"

// NCAAScoreboard is a ScoreboardFetcher that pulls daily scoreboards
// through the NCAA proxy. The response shape matches NCAA.com's
// internal JSON: each game is wrapped in a `game` object with
// away/home sides keyed by `score`, `names.seo`, `names.short`,
// `winner`, `rank`, and conferences.
type NCAAScoreboard struct {
	base   string
	client *http.Client
}

// NewNCAAScoreboard builds a scraper. Pass "" for base to use
// DefaultNCAABaseURL. A nil client defaults to http.DefaultClient with
// a 15-second timeout.
func NewNCAAScoreboard(base string, client *http.Client) *NCAAScoreboard {
	if base == "" {
		base = DefaultNCAABaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &NCAAScoreboard{base: strings.TrimRight(base, "/"), client: client}
}

// FetchDaily fetches the scoreboard for one date. Returns a wrapped
// AppError on upstream failure or malformed response.
func (s *NCAAScoreboard) FetchDaily(ctx context.Context, sport domain.SportID, div domain.Division, date time.Time) ([]domain.Game, error) {
	url := fmt.Sprintf("%s/scoreboard/%s/%s/%s", s.base, sport, div, date.Format("2006/01/02"))
	payload, empty, err := s.fetchScoreboardPayload(ctx, url)
	if err != nil {
		return nil, err
	}
	if empty {
		return []domain.Game{}, nil
	}

	games := make([]domain.Game, 0, len(payload.Games))
	for _, wrap := range payload.Games {
		g, ok := convertGame(wrap.Game, date)
		if !ok {
			continue
		}
		games = append(games, g)
	}
	return games, nil
}

// fetchScoreboardPayload performs the HTTP round-trip and decodes the
// upstream response. The `empty` return is true when the upstream
// treated the date as valid but reports no games (observed as a 404
// early in the season).
func (s *NCAAScoreboard) fetchScoreboardPayload(ctx context.Context, url string) (scoreboardPayload, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return scoreboardPayload{}, false, apperrors.Wrap(apperrors.ErrCodeInternal, "scraper: build request", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "college-sports-api/0.1")

	resp, err := s.client.Do(req)
	if err != nil {
		return scoreboardPayload{}, false, apperrors.Wrap(apperrors.ErrCodeUnavailable, "scraper: upstream fetch", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusNotFound:
		return scoreboardPayload{}, true, nil
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return scoreboardPayload{}, false, apperrors.New(apperrors.ErrCodeUnavailable,
			fmt.Sprintf("scraper: upstream status %d: %s", resp.StatusCode, string(body)))
	}

	var payload scoreboardPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return scoreboardPayload{}, false, apperrors.Wrap(apperrors.ErrCodeUnavailable, "scraper: decode scoreboard", err)
	}
	return payload, false, nil
}

// scoreboardPayload mirrors the proxy's response shape.
type scoreboardPayload struct {
	Games []struct {
		Game ncaaGame `json:"game"`
	} `json:"games"`
}

type ncaaGame struct {
	GameID         string   `json:"gameID"`
	Away           ncaaTeam `json:"away"`
	Home           ncaaTeam `json:"home"`
	FinalMessage   string   `json:"finalMessage"`
	StartTime      string   `json:"startTime"`
	StartTimeEpoch string   `json:"startTimeEpoch"`
	GameState      string   `json:"gameState"`
	StartDate      string   `json:"startDate"`
	URL            string   `json:"url"`
	Network        string   `json:"network"`
	BracketRound   string   `json:"bracketRound"`
}

type ncaaTeam struct {
	Score string `json:"score"`
	Names struct {
		Char6 string `json:"char6"`
		Short string `json:"short"`
		Seo   string `json:"seo"`
		Full  string `json:"full"`
	} `json:"names"`
	Winner      bool   `json:"winner"`
	Rank        string `json:"rank"`
	Conferences []struct {
		ConferenceName string `json:"conferenceName"`
		ConferenceSeo  string `json:"conferenceSeo"`
	} `json:"conferences"`
}
