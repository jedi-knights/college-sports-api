# College Sports API

> Fast, open NCAA scoreboard, schedule, and RPI data derived from NCAA.com.

![CI](https://github.com/jedi-knights/college-sports-api/actions/workflows/ci.yml/badge.svg)
![Deploy](https://github.com/jedi-knights/college-sports-api/actions/workflows/deploy.yml/badge.svg)
![Docker](https://github.com/jedi-knights/college-sports-api/actions/workflows/docker.yml/badge.svg)
[![codecov](https://codecov.io/gh/jedi-knights/college-sports-api/branch/main/graph/badge.svg)](https://codecov.io/gh/jedi-knights/college-sports-api)
[![Go Reference](https://pkg.go.dev/badge/github.com/jedi-knights/college-sports-api.svg)](https://pkg.go.dev/github.com/jedi-knights/college-sports-api)
[![License: MIT](https://img.shields.io/badge/license-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.26-blue.svg)](https://go.dev/dl/)

A sport-pluggable Go HTTP API that scrapes NCAA.com and exposes
scoreboards, team schedules, and computed metrics (standings, RPI)
behind a Redis-backed cache.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
- [Examples](#examples)
- [Configuration](#configuration)
- [Development](#development)
- [Deployment](#deployment)
- [Architecture](#architecture)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)
- [Acknowledgements](#acknowledgements)

## Overview

The NCAA publishes live scoreboards, standings, and schedule data for
every college sport at [ncaa.com][ncaa] — but there is no public,
documented API. The site's own data layer is a Next.js SSR shell that
reads from an unlisted GraphQL endpoint using persisted-query hashes
that change without notice.

Community-maintained proxies like [`ncaa-api.henrygd.me`][henrygd]
have filled the gap for years. They do a very good job of exposing
the **daily scoreboard** across ~25 sports. What they don't do:

- Serve **per-team season schedules** — the single most-requested
  endpoint, missing from every proxy today.
- Compute **domain metrics** (standings, RPI, strength of schedule).
- **Filter correctness** — scoreboards include exhibition and
  intra-squad games alongside real contests; downstream consumers
  have to recognize and strip them.
- Insulate consumers from upstream fragility — a GraphQL-hash
  rotation at NCAA takes every community proxy offline until a human
  updates them.

Rolling your own is not cheap. You need to track NCAA.com's Next.js
build ID, discover and refresh persisted-query hashes, handle three
different URL shapes for current / recent / historical data, cache
responses correctly, parse team names consistently across divisions,
and operate the thing. That's the work this project absorbs:

```bash
curl -s "https://jk-college-sports-api.fly.dev/v1/scoreboard/soccer-women/d1/2026/10/11" \
  | jq '.count, .games[0].home.short, .games[0].away.short'
```

Returns the day's D1 women's soccer games with normalized `home`
and `away` sides, scores, state (`final` / `live` / `scheduled`),
NCAA team slugs for every program, and the `is_exhibition` flag the
server has already applied the filter against.

## Features

- **Daily scoreboards** for all college soccer (men's and women's,
  D1 / D2 / D3) at `/v1/scoreboard/{sport}/{division}/{yyyy}/{mm}/{dd}`.
- **Sport-pluggable architecture** — new sports slot in via one
  `Register(reg)` call, no god interface.
- **Redis-backed cache** with sensible per-endpoint TTLs. Works
  against local Redis in development and Fly-managed Upstash Redis
  in production — same wire protocol, same client.
- **Correctness filters** — `exclude_exhibitions` on every endpoint
  that returns games (default `true`).
- **Operational endpoints** — `/healthz`, `/readyz`, `/metrics`
  (Prometheus), `/openapi.yaml`.
- **Hexagonal / clean architecture** — adapters depend on ports,
  ports depend on domain, domain depends on nothing.
- **Published OpenAPI 3.1 spec** that doubles as the test oracle.
- **Fly.io deployable** out of the box — `fly launch` with the
  included `deploy/fly.toml` and `deploy/Dockerfile`.

Future phases add: GraphQL hash self-heal, legacy-REST fallback,
per-team schedules, team metadata, computed standings and RPI,
Swagger UI at `/docs`.

## Requirements

| Tool    | Version | Why                               |
|---------|---------|-----------------------------------|
| Go      | 1.26+   | The server toolchain.             |
| Docker  | 24+     | Local Redis + the API container.  |
| Redis   | 7+      | Only needed if not using Docker.  |
| Task    | 3+      | Runs `Taskfile.yml` targets.      |
| flyctl  | latest  | Deploy to Fly.io.                 |

Install the Go toolchain from [go.dev/dl][godl], Task via
[`brew install go-task`][gotask], and flyctl from
[fly.io/docs/flyctl/install][flyctl].

## Installation

Three equivalent paths. Pick based on what you need:

**Docker (recommended for users):**

```bash
docker run --rm -p 8080:8080 \
  -e CSA_CACHE_BACKEND=memory \
  ghcr.io/jedi-knights/college-sports-api:latest serve
```

**Go install:**

```bash
go install github.com/jedi-knights/college-sports-api/cmd/server@latest
college-sports-api serve
```

**From source (recommended for contributors):**

```bash
git clone https://github.com/jedi-knights/college-sports-api.git
cd college-sports-api
task dev           # starts the API + Redis in docker-compose
```

## Usage

Start the server (any installation path above). It listens on
`:8080` by default.

```bash
curl -s "http://localhost:8080/v1/scoreboard/soccer-women/d1/2026/10/11" | jq .
```

```json
{
  "sport": "soccer-women",
  "division": "d1",
  "date": "2026-10-11",
  "count": 98,
  "games": [
    {
      "id": "6609557",
      "date": "2026-10-11",
      "start_time": "2026-10-11T15:00:00Z",
      "state": "final",
      "home": { "seo": "howard",   "short": "Howard",   "score": 2 },
      "away": { "seo": "le-moyne", "short": "Le Moyne", "score": 3, "winner": true }
    }
    // ...
  ]
}
```

Hit `/openapi.yaml` for the complete schema or `/metrics` for the
Prometheus scrape target.

## Examples

### 1. Yesterday's D1 women's soccer scoreboard

```bash
DATE=$(date -v-1d +%Y/%m/%d)
curl -s "http://localhost:8080/v1/scoreboard/soccer-women/d1/${DATE}" | jq '.count'
```

### 2. One day of D1 men's soccer

```bash
curl -s "http://localhost:8080/v1/scoreboard/soccer-men/d1/2026/10/11" \
  | jq '.games[] | select(.state == "final") | {home: .home.short, away: .away.short, final: "\(.home.score)-\(.away.score)"}'
```

### 3. Readiness check against Redis

```bash
curl -i http://localhost:8080/readyz
# HTTP/1.1 200 OK on green; 503 when Redis is unreachable.
```

## Configuration

Every YAML key is overridable by an env var with the `CSA_` prefix
and dot→underscore replacement. See `config/config.example.yaml` for
the complete annotated file.

| YAML key                | Env var                   | Default                      | Effect                                       |
|-------------------------|---------------------------|------------------------------|----------------------------------------------|
| `server.port`           | `CSA_SERVER_PORT`         | `8080`                       | TCP port the HTTP server listens on.         |
| `log.level`             | `CSA_LOG_LEVEL`           | `info`                       | `debug` \| `info` \| `warn` \| `error`.      |
| `log.format`            | `CSA_LOG_FORMAT`          | `json`                       | `json` \| `text`.                            |
| `cache.backend`         | `CSA_CACHE_BACKEND`       | `memory`                     | `redis` \| `memory`.                         |
| `cache.redis_url`       | `CSA_REDIS_URL`           | *(empty)*                    | `redis://[:password@]host:port[/db]`.        |
| `scraper.base_url`      | `CSA_SCRAPER_BASE_URL`    | *(built-in)*                 | Override the upstream data source.           |
| `outbound.rate_limit`   | `CSA_OUTBOUND_RATE_LIMIT` | `3.0`                        | Requests/sec allowed toward NCAA upstreams.  |

## Development

Everything is a `Taskfile.yml` target. Run `task` for the current
list. The common ones:

```bash
task dev               # start API + Redis in docker-compose
task run               # run the server directly against config.example.yaml
task test              # unit tests, race detector, build tag `unit`
task test-integration  # hits real NCAA upstream
task lint              # golangci-lint
task openapi-check     # lint api/openapi/openapi.yaml with redocly
task docker            # build the production image
task tidy              # go mod tidy && go mod verify
task ci                # tidy → lint → test → build
```

The project targets Go 1.26 and uses the same conventions as every
other service in the `jedi-knights` organization:

- Hexagonal layout (`internal/{domain,ports,adapters,container,config,sports}`).
- Stdlib `net/http` — no gin / echo / chi.
- `go-platform` for shared cross-cutting (`apperrors`, `httpmw`,
  `httpserver`, `httputil`).
- `go-logging` as the slog wrapper.
- `.golangci.yml` with `max-complexity: 7` and `misspell.locale: US`.
- Table-driven tests behind `//go:build unit`.
- Conventional Commits with package-scoped messages
  (`feat(scraper): …`, `fix(cache): …`).

## Deployment

The repository ships with a production-ready
`deploy/Dockerfile` (multi-stage, distroless runtime) and
`deploy/fly.toml` configured for Fly.io's shared-cpu-1x free tier.

```bash
# One-time: create the app and attach a managed Redis.
flyctl apps create jk-college-sports-api
flyctl redis create                     # creates an Upstash Redis instance
flyctl secrets set CSA_REDIS_URL=redis://... # value from `fly redis status`

# Deploy. The GitHub Actions `deploy.yml` workflow runs this on every push to main.
task fly-deploy
```

Upstash's Fly-managed free tier gives 250 MB of storage, 500K
commands/month, and 10 GB of bandwidth. For the API's cache shape
(TTL-dominated, singleflight-coalesced) that's generous — the plan
pins a `cache_commands_total` metric that alerts at 80% of the free
quota before overage pricing kicks in.

## Architecture

```
cmd/
  server/              Cobra entrypoint, container boot, graceful shutdown
internal/
  config/              Viper config loader (YAML + env)
  container/           Capability registry — the sport-pluggability seam
  domain/              Sport-agnostic types (Game, Team, Sport, Division)
  ports/               Interfaces consumed by inbound and outbound adapters
  adapters/
    inbound/http/      Router, handlers, health probes, OpenAPI, metrics
    outbound/
      scraper/         NCAA scoreboard fetchers (ncaa_scoreboard.go; graphql.go in phase 2)
      cache/           Cache interface + Redis + in-memory implementations
  sports/
    soccer/            Men's and women's soccer sport descriptors + registration
api/openapi/           Hand-written OpenAPI 3.1 spec
config/                Example YAML + GraphQL-hash seed
deploy/                Dockerfile, docker-compose, fly.toml
```

Dependency arrow: adapters → ports → domain. The domain package
has no imports from the rest of the codebase, by design.

## Contributing

Pull requests are welcome. The project follows the standard
`jedi-knights` contribution rules:

1. Fork and create a branch — one `<type>(<scope>): …` concern per PR.
2. Run `task ci` locally before pushing.
3. Add a table-driven unit test for new logic.
4. If your change touches a public endpoint shape, update
   `api/openapi/openapi.yaml` in the same change and run
   `task openapi-check`.
5. Commit messages follow [Conventional Commits][cc]. Scope is the
   package name (`feat(container): …`, `fix(scraper): …`).

For questions or design discussions, open an issue before a large PR.

## Security

See [SECURITY.md](SECURITY.md) for the vulnerability-reporting
policy.

## License

[MIT](LICENSE) © Omar Crosby

## Acknowledgements

[`henrygd/ncaa-api`][henrygd-src] is the TypeScript proxy whose
surface this project started from. Its endpoint shapes and the
upstream data patterns it decoded saved substantial reverse-engineering
work.

[ncaa]: https://www.ncaa.com
[henrygd]: https://ncaa-api.henrygd.me
[henrygd-src]: https://github.com/henrygd/ncaa-api
[godl]: https://go.dev/dl/
[gotask]: https://taskfile.dev/installation/
[flyctl]: https://fly.io/docs/flyctl/install/
[cc]: https://www.conventionalcommits.org/en/v1.0.0/
