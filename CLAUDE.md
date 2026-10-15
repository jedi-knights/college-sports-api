# CLAUDE.md — college-sports-api

Repo-specific guidance for Claude working inside this project.

## What this service does

A sport-pluggable Go HTTP API that mirrors the NCAA.com scoreboard
surface (initially men's and women's soccer, D1/D2/D3) and adds
per-team schedules, computed metrics, and correctness filters.
Public-facing; deployed to Fly.io with a Redis cache.

## Architecture invariants

**Hexagonal.** `adapters → ports → domain`. The domain package has
no imports from anywhere else in the repo. Never let an adapter type
leak into `domain/` — if it feels tempting, you've found a missing
port.

**Capability maps, not a god Sport interface.** `container.Registry`
holds separate typed maps for each capability (ScoreboardFetcher,
TeamScheduleFetcher, etc.). A sport without a given capability is
simply absent from the map; the handler layer returns 404 at the
map lookup. Never introduce a `nil`-returning method on `Sport` to
represent "unsupported".

**Stdlib `net/http` only.** No gin, echo, chi, fiber. We get method+path
pattern matching from Go 1.22+'s stdlib router.

**Error flow.** Internal failures are `apperrors.AppError`. Handlers
render them via `httputil.WriteError`. Never shape user-facing JSON
by hand — the renderer owns the shape.

**Cache is the critical path.** Every endpoint that fetches upstream
MUST read-through the cache. Direct scraper calls without a cache in
front are a performance regression (and a cost regression on the
Upstash free tier). The one exception is `graphql_discovery` (Phase
2), which bootstraps the cache rather than reading from it.

## Lint invariants

`.golangci.yml` sets `cyclop.max-complexity: 7` and
`gocyclo.min-complexity: 7`. If a function hits 8+, the fix is **not**
`//nolint:cyclop` — split the function. The ones I've already fixed
this way: `scoreboardHandler.ServeHTTP`, `NCAAScoreboard.FetchDaily`,
`parseScoreboardPath`.

`misspell.locale: US` is strict. Common failure: `cancelled` must be
`canceled`. Grep before writing new state names.

## When adding a new sport

1. Create `internal/sports/<sport>/sport.go` with a descriptor type
   per side (`Men`, `Women`), each implementing `domain.Sport`.
2. Register it in `cmd/server/main.go::buildRegistry` with its sport
   ID plus any capability it supports.
3. Add the sport ID constant to `internal/domain/sport.go`.
4. Add the enum value to `api/openapi/openapi.yaml`'s `Sport` parameter.

## When adding a new endpoint

1. Add the port interface to `internal/ports/<capability>.go`.
2. Add the capability map + register method to
   `internal/container/registry.go`.
3. Add the handler to `internal/adapters/inbound/http/<name>.go`.
   Pattern-match the scoreboard handler's cache read-through
   structure.
4. Register the route in `internal/adapters/inbound/http/router.go`.
5. Document in `api/openapi/openapi.yaml` **in the same PR**.

## Phase status (as of plan v1)

- **Phase 1 — DONE**: scoreboard endpoint for soccer-men/women,
  Redis + memory cache, Fly.io deployable, OpenAPI, CI, README.
  Upstream is `ncaa-api.henrygd.me` as a bootstrap source (clearly
  documented in `scraper/ncaa_scoreboard.go`).
- **Phase 2**: GraphQL hash self-heal, legacy REST fallback,
  Composite scraper. Replaces henrygd dependency.
- **Phase 3**: per-team schedule, team metadata, Swagger UI at `/docs`.
- **Phase 4**: standings, rankings, computed RPI. Port
  `wsoccer-stats/src-tauri/src/standings.rs::compute_rpi_full` to
  Go verbatim.

The full plan is in
`~/.claude/plans/ok-examine-ncaa-api-henrygd-me-i-silly-owl.md`.
