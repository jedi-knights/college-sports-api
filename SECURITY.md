# Security policy

## Reporting a vulnerability

Please report security issues privately via GitHub's
["Report a vulnerability" workflow][gh-advisory] on this repository.
Do not open a public issue for anything that could be exploited.

Expected response time: within 72 hours.

## Scope

- The HTTP server and its handlers under `cmd/` and `internal/`.
- The Dockerfile and `deploy/` configuration.
- The published OpenAPI contract.

Out of scope:

- Vulnerabilities in NCAA.com or any upstream data source — those
  are owned by their respective publishers.
- Issues in third-party dependencies without a demonstrable impact
  on this service. Report those to the upstream project; we'll bump
  pins once a patched version is released.

## Supported versions

Only the `main` branch and the most recent minor release receive
security updates.

[gh-advisory]: https://github.com/jedi-knights/college-sports-api/security/advisories/new
