package http

import (
	"net/http"
	"os"

	"github.com/jedi-knights/go-platform/apperrors"
	"github.com/jedi-knights/go-platform/httputil"
)

// openAPISearchPaths is the lookup order for the published spec. The
// first readable file wins. Order matches the Dockerfile and
// config-example conventions: baked-in path first, then repo-relative
// for `task run`.
var openAPISearchPaths = []string{
	"/etc/college-sports-api/openapi.yaml",
	"api/openapi/openapi.yaml",
}

// openAPIHandler serves the OpenAPI 3.1 spec as `application/yaml`.
// Browsers and tools can hit `/openapi.yaml` without special
// `Accept:` negotiation.
func openAPIHandler(w http.ResponseWriter, _ *http.Request) {
	for _, p := range openAPISearchPaths {
		body, err := os.ReadFile(p) // #nosec G304 — p is from a static allowlist above
		if err != nil {
			continue
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return
	}
	httputil.WriteError(w, apperrors.New(apperrors.ErrCodeNotFound, "openapi spec not found on this instance"))
}
