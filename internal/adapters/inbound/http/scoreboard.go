package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jedi-knights/go-platform/apperrors"
	"github.com/jedi-knights/go-platform/httputil"

	"github.com/jedi-knights/college-sports-api/internal/container"
	"github.com/jedi-knights/college-sports-api/internal/domain"
	"github.com/jedi-knights/college-sports-api/internal/ports"
)

// scoreboardTTL is the cache lifetime for a day's scoreboard. 60s
// keeps live-game scoreboards snappy while absorbing most refresh
// traffic. Historical dates could get a longer TTL once we land the
// per-date freshness tier (plan Phase 2).
const scoreboardTTL = 60 * time.Second

type scoreboardHandler struct {
	registry *container.Registry
	cache    ports.Cache
}

type scoreboardResponse struct {
	Sport    domain.SportID  `json:"sport"`
	Division domain.Division `json:"division"`
	Date     string          `json:"date"`
	Count    int             `json:"count"`
	Games    []domain.Game   `json:"games"`
}

func (h *scoreboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sport, div, date, err := parseScoreboardPath(r)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	fetcher, ok := h.registry.ScoreboardFetcher(sport)
	if !ok {
		httputil.WriteError(w, apperrors.New(apperrors.ErrCodeNotFound,
			fmt.Sprintf("sport %q has no scoreboard fetcher", sport)))
		return
	}

	cacheKey := scoreboardCacheKey(sport, div, date)
	if cached, hit := h.readCache(r.Context(), cacheKey); hit {
		writeRaw(w, http.StatusOK, cached)
		return
	}

	games, err := fetcher.FetchDaily(r.Context(), sport, div, date)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	resp := scoreboardResponse{
		Sport:    sport,
		Division: div,
		Date:     date.Format("2006-01-02"),
		Count:    len(games),
		Games:    games,
	}
	payload, mErr := json.Marshal(resp)
	if mErr != nil {
		httputil.WriteError(w, apperrors.Wrap(apperrors.ErrCodeInternal, "scoreboard: marshal response", mErr))
		return
	}
	// Best effort — a cache failure must not fail the request.
	_ = h.cache.Set(r.Context(), cacheKey, payload, scoreboardTTL)

	writeRaw(w, http.StatusOK, payload)
}

func (h *scoreboardHandler) readCache(ctx context.Context, key string) ([]byte, bool) {
	v, err := h.cache.Get(ctx, key)
	if err == nil {
		return v, true
	}
	if errors.Is(err, ports.ErrCacheMiss) {
		return nil, false
	}
	// Any other cache error is treated as a miss; the upstream fetch
	// will handle it. The logger middleware captures the trace, and
	// the cache metric records the error separately.
	return nil, false
}

func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func scoreboardCacheKey(sport domain.SportID, div domain.Division, date time.Time) string {
	return fmt.Sprintf("csa:v1:scoreboard:%s:%s:%s", sport, div, date.Format("2006-01-02"))
}

func parseScoreboardPath(r *http.Request) (domain.SportID, domain.Division, time.Time, error) {
	sport := domain.SportID(r.PathValue("sport"))
	div := domain.Division(r.PathValue("division"))
	year, err := parseIntField(r, "year", 1900, 2100)
	if err != nil {
		return "", "", time.Time{}, err
	}
	month, err := parseIntField(r, "month", 1, 12)
	if err != nil {
		return "", "", time.Time{}, err
	}
	day, err := parseIntField(r, "day", 1, 31)
	if err != nil {
		return "", "", time.Time{}, err
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return sport, div, date, nil
}

// parseIntField reads an integer path parameter and asserts it is
// within [min, max] (inclusive). Returns a BAD_REQUEST AppError on
// any failure.
func parseIntField(r *http.Request, name string, minVal, maxVal int) (int, error) {
	raw := r.PathValue(name)
	n, err := strconv.Atoi(raw)
	if err != nil || n < minVal || n > maxVal {
		return 0, apperrors.New(apperrors.ErrCodeBadRequest,
			fmt.Sprintf("invalid %s path parameter: %q", name, raw))
	}
	return n, nil
}
