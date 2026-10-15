# Contributing to college-sports-api

Thanks for your interest. This document is the short version; the
full flow lives in [README.md](README.md#contributing).

## Development loop

```bash
git clone https://github.com/jedi-knights/college-sports-api.git
cd college-sports-api
task dev        # starts API + Redis in docker-compose
task ci         # tidy → lint → test → build (run before pushing)
```

## Pull-request checklist

- [ ] Branch is a single `<type>(<scope>): …` concern. Split if the
      description contains "and".
- [ ] `task ci` passes locally.
- [ ] Table-driven unit test(s) added for new logic. Build tag
      `//go:build unit`.
- [ ] `api/openapi/openapi.yaml` updated if a public endpoint shape
      changed, and `task openapi-check` passes.
- [ ] Commit messages follow [Conventional Commits][cc] with the
      package name as scope (`feat(scraper): …`, `fix(cache): …`).
- [ ] No new direct NCAA scraping without a cache in front — see the
      `cache.Cache` port.

## Reporting bugs

Open a [GitHub issue][issues] with:

- Expected vs actual behavior.
- A minimal reproduction (ideally a `curl` command).
- Server logs if available (the structured JSON format at `info`+
  level is enough).

## Discussing design

For anything larger than a single-file change, open an issue first
and tag it `design`. The hexagonal boundaries in this project are
load-bearing — a reviewer will push back on a PR that bypasses
`ports`.

## Code style

- Match `.golangci.yml` (max-complexity 7, misspell `US`).
- US English in all identifiers, comments, and user-facing strings.
- Comments explain **why**, not **what**. Keep them one line unless
  a hidden invariant genuinely needs more.

[cc]: https://www.conventionalcommits.org/en/v1.0.0/
[issues]: https://github.com/jedi-knights/college-sports-api/issues
