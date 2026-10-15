package http

import (
	"context"
	"net/http"
	"time"

	"github.com/jedi-knights/go-platform/httputil"

	"github.com/jedi-knights/college-sports-api/internal/ports"
)

// readyzPingTimeout caps the cache ping so a hung backend doesn't let
// Fly's health check time out on the proxy side. Must stay under the
// http_service check's `timeout` in deploy/fly.toml (3s).
const readyzPingTimeout = 2 * time.Second

// healthHandler serves /healthz (liveness) and /readyz (readiness).
// Liveness is a trivial 200; readiness pings the cache.
type healthHandler struct {
	cache  ports.Cache
	logger logger
}

// logger is the subset of go-logging the handler needs. Kept narrow
// so tests can inject a stub without pulling in the full logging
// surface.
type logger interface {
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type healthResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (h *healthHandler) live(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (h *healthHandler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyzPingTimeout)
	defer cancel()
	if err := h.cache.Ping(ctx); err != nil {
		// Log the raw error so operators can grep `fly logs` for the
		// underlying go-redis / network cause. The response body
		// surfaces the error type only — never credentials or
		// upstream hostnames.
		if h.logger != nil {
			h.logger.ErrorContext(ctx, "readyz cache ping failed", "error", err.Error())
		}
		httputil.WriteJSON(w, http.StatusServiceUnavailable, healthResponse{
			Status: "cache unreachable",
			Error:  err.Error(),
		})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
