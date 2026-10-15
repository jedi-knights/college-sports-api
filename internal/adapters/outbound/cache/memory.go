// Package cache contains Cache port implementations. The redis backend
// is the production path; the memory backend is a dev/unit-test
// fallback with the same semantics (TTL, miss returns ports.ErrCacheMiss).
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/jedi-knights/college-sports-api/internal/ports"
)

// Memory is an in-process Cache implementation backed by a sync.Map
// with a background janitor that evicts expired keys. Safe for
// concurrent use.
//
// Intended for unit tests and the `task run` (no-Docker) workflow.
// Prefer Redis for anything multi-process or long-lived.
type Memory struct {
	entries sync.Map // key string → entry
	done    chan struct{}
}

type memoryEntry struct {
	value   []byte
	expires time.Time // zero == no expiry
}

// NewMemory returns a Memory cache and starts the janitor. Call Close
// to stop the janitor; otherwise it runs for the process lifetime.
func NewMemory(janitorInterval time.Duration) *Memory {
	if janitorInterval <= 0 {
		janitorInterval = time.Minute
	}
	m := &Memory{done: make(chan struct{})}
	go m.janitor(janitorInterval)
	return m
}

// Close stops the janitor goroutine. Idempotent.
func (m *Memory) Close() {
	select {
	case <-m.done:
		return
	default:
		close(m.done)
	}
}

// Get returns the cached value, or ports.ErrCacheMiss when absent or
// expired. Expired entries are evicted lazily on read in addition to
// the periodic janitor sweep.
func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := m.entries.Load(key)
	if !ok {
		return nil, ports.ErrCacheMiss
	}
	entry, _ := v.(memoryEntry)
	if !entry.expires.IsZero() && time.Now().After(entry.expires) {
		m.entries.Delete(key)
		return nil, ports.ErrCacheMiss
	}
	// Return a copy so the caller can't mutate the stored slice.
	out := make([]byte, len(entry.value))
	copy(out, entry.value)
	return out, nil
}

// Set stores value under key with the given TTL. Zero or negative TTL
// stores a never-expiring entry.
func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	stored := make([]byte, len(value))
	copy(stored, value)
	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	m.entries.Store(key, memoryEntry{value: stored, expires: expires})
	return nil
}

// Delete removes zero or more keys. Missing keys are not an error.
func (m *Memory) Delete(_ context.Context, keys ...string) error {
	for _, k := range keys {
		m.entries.Delete(k)
	}
	return nil
}

// Ping always returns nil — the in-process cache is always reachable.
func (m *Memory) Ping(_ context.Context) error { return nil }

func (m *Memory) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-m.done:
			return
		case now := <-ticker.C:
			m.entries.Range(func(k, v any) bool {
				entry, _ := v.(memoryEntry)
				if !entry.expires.IsZero() && now.After(entry.expires) {
					m.entries.Delete(k)
				}
				return true
			})
		}
	}
}
