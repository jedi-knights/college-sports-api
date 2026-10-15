package ports

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned by Cache.Get when the key is not present.
// This is not an error condition in the apperrors sense — callers
// treat it as "fetch from upstream" and should not wrap it.
var ErrCacheMiss = errors.New("cache: miss")

// Cache is the storage abstraction behind every endpoint. Keys are
// namespaced (`csa:v1:scoreboard:soccer-women:d1:2026-10-11`), values
// are opaque JSON blobs the handler layer serializes before Set and
// decodes after Get.
//
// The interface is intentionally narrow: Get, Set, Delete. Anything
// beyond that (TTL extensions, cache-aside wrappers, singleflight
// coalescing) lives in composition above this interface.
type Cache interface {
	// Get returns the cached value for key. Returns ErrCacheMiss when
	// the key is absent — not a wrapped AppError, because a miss is
	// routine control flow.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores value under key with the given TTL. A zero TTL means
	// "never expire" — implementations should refuse this at the API
	// boundary rather than silently keep keys forever.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes one or more keys. Deleting a missing key is not
	// an error.
	Delete(ctx context.Context, keys ...string) error

	// Ping checks that the backing store is reachable. Used by the
	// readiness probe. Returns a wrapped error on failure.
	Ping(ctx context.Context) error
}
