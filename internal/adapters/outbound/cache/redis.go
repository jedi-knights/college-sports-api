package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jedi-knights/go-platform/apperrors"
	"github.com/redis/go-redis/v9"

	"github.com/jedi-knights/college-sports-api/internal/ports"
)

// Redis is a Cache implementation backed by a Redis-protocol server
// (Redis locally or Upstash on Fly.io). The same binary wire format
// means one adapter covers both deployments.
type Redis struct {
	client *redis.Client
}

// NewRedis builds a Redis cache from a go-redis URL
// (`redis://[:password@]host:port[/db]`). The returned Redis is safe
// for concurrent use. Close it on shutdown to release the connection
// pool.
func NewRedis(url string) (*Redis, error) {
	if url == "" {
		return nil, apperrors.New(apperrors.ErrCodeBadRequest, "cache: redis URL is required")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeBadRequest, "cache: invalid redis URL", err)
	}
	return &Redis{client: redis.NewClient(opts)}, nil
}

// Close releases the connection pool.
func (r *Redis) Close() error { return r.client.Close() }

// Get returns the cached bytes, or ports.ErrCacheMiss when the key is
// absent. All other failures bubble up as a wrapped AppError.
func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := r.client.Get(ctx, key).Bytes()
	if err == nil {
		return out, nil
	}
	if errors.Is(err, redis.Nil) {
		return nil, ports.ErrCacheMiss
	}
	return nil, apperrors.Wrap(apperrors.ErrCodeUnavailable, "cache: get failed", err)
}

// Set stores value under key with the given TTL. The TTL must be
// positive — the public API never wants eternal keys in the shared
// store.
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return apperrors.New(apperrors.ErrCodeBadRequest, fmt.Sprintf("cache: ttl must be positive, got %s", ttl))
	}
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeUnavailable, "cache: set failed", err)
	}
	return nil
}

// Delete removes zero or more keys. Missing keys are not an error.
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeUnavailable, "cache: delete failed", err)
	}
	return nil
}

// Ping checks that the server is reachable. Used by the readiness
// probe.
func (r *Redis) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeUnavailable, "cache: redis ping failed", err)
	}
	return nil
}
