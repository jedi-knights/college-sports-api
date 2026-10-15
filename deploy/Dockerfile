# syntax=docker/dockerfile:1

# ----- Builder -------------------------------------------------------------
# Pinned to the same Go version as go.mod / CI (.github/workflows/ci.yml).
FROM golang:1.26 AS builder

WORKDIR /src

# Download modules first so the layer is cached when only source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Version metadata injected via ldflags (see cmd/server/version.go). Passed
# from fly.toml [build.args] or the CD workflow; defaults keep local builds
# working.
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# CGO disabled → a fully static binary that runs on distroless/static.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o /college-sports-api ./cmd/server

# ----- Runtime -------------------------------------------------------------
# distroless/static ships CA certificates (needed for HTTPS to NCAA upstreams)
# and runs as an unprivileged user. Nothing else is in the image.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /college-sports-api /college-sports-api
# Baked-in configuration search order (see internal/config/config.go).
COPY config/config.example.yaml /etc/college-sports-api/config.yaml
COPY config/graphql-hashes.json /etc/college-sports-api/graphql-hashes.json
COPY api/openapi/openapi.yaml /etc/college-sports-api/openapi.yaml

EXPOSE 8080

ENTRYPOINT ["/college-sports-api", "serve"]
