// Package http wires the inbound HTTP surface: routing, handlers,
// middleware, error rendering, OpenAPI serving, Prometheus metrics,
// and health probes.
package http

import (
	"context"
	"net/http"

	"github.com/jedi-knights/go-platform/httpmw"
	"github.com/jedi-knights/go-platform/httputil"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jedi-knights/college-sports-api/internal/container"
	"github.com/jedi-knights/college-sports-api/internal/ports"
)

// Deps are everything the router needs. Pass them in from the
// composition root in cmd/server.
type Deps struct {
	Logger   httpmw.Logger
	Registry *container.Registry
	Cache    ports.Cache
}

// NewRouter builds the HTTP handler for the public API. The returned
// handler is wrapped with the platform's standard middleware stack
// (request ID → trace ID → recovery → logging), so callers hand this
// straight to httpserver.New.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()

	scoreboard := &scoreboardHandler{registry: d.Registry, cache: d.Cache}
	mux.Handle("GET /v1/scoreboard/{sport}/{division}/{year}/{month}/{day}", scoreboard)

	health := &healthHandler{cache: d.Cache}
	mux.HandleFunc("GET /healthz", health.live)
	mux.HandleFunc("GET /readyz", health.ready)

	mux.Handle("GET /metrics", promhttp.Handler())

	mux.HandleFunc("GET /openapi.yaml", openAPIHandler)

	// Catch-all 404 → structured JSON instead of the stdlib default.
	mux.HandleFunc("/", notFoundHandler)

	return httpmw.Stack(d.Logger, httpmw.WithSkipLogPaths("/healthz", "/readyz", "/metrics"))(mux)
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusNotFound, httputil.ErrorResponse{
		Error: "route not found",
		Code:  "NOT_FOUND",
	})
}

// ensure context is used in some file (keeps imports stable across edits).
var _ = context.Background
