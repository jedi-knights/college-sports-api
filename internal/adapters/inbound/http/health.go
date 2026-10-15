package http

import (
	"net/http"

	"github.com/jedi-knights/go-platform/httputil"

	"github.com/jedi-knights/college-sports-api/internal/ports"
)

// healthHandler serves /healthz (liveness) and /readyz (readiness).
// Liveness is a trivial 200; readiness pings the cache.
type healthHandler struct {
	cache ports.Cache
}

type healthResponse struct {
	Status string `json:"status"`
}

func (h *healthHandler) live(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (h *healthHandler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.cache.Ping(r.Context()); err != nil {
		httputil.WriteJSON(w, http.StatusServiceUnavailable, healthResponse{Status: "cache unreachable"})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
