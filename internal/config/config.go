// Package config loads server configuration from a YAML file,
// environment variables, and sensible defaults — in that precedence
// (defaults → file → env). All env vars share the `CSA_` prefix.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the full runtime configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Log      LogConfig      `mapstructure:"log"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Scraper  ScraperConfig  `mapstructure:"scraper"`
	Outbound OutboundConfig `mapstructure:"outbound"`
}

// ServerConfig configures the HTTP listener.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// LogConfig configures slog output.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// CacheConfig configures the Cache adapter.
type CacheConfig struct {
	// Backend is one of "redis" or "memory".
	Backend  string `mapstructure:"backend"`
	RedisURL string `mapstructure:"redis_url"`
}

// ScraperConfig configures the NCAA scoreboard scraper.
type ScraperConfig struct {
	// BaseURL overrides the upstream data source. Empty uses the
	// built-in default.
	BaseURL string `mapstructure:"base_url"`
}

// OutboundConfig configures the global outbound rate limiter.
type OutboundConfig struct {
	// RateLimit is commands/sec allowed toward NCAA upstreams.
	RateLimit float64 `mapstructure:"rate_limit"`
}

// Load reads configuration from the given path (optional — empty
// means "no file") then overlays env vars prefixed with `CSA_`.
// Defaults are applied for anything neither source specifies.
func Load(path string) (Config, error) {
	v := viper.New()
	setDefaults(v)
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("config: read %s: %w", path, err)
		}
	}
	v.SetEnvPrefix("CSA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicit short-form aliases for env vars we document in the
	// README. Without these, `cache.redis_url` would only match
	// `CSA_CACHE_REDIS_URL` under AutomaticEnv — and Fly secrets like
	// `CSA_REDIS_URL` (the shape every doc, docker-compose, and
	// deploy guide uses) would silently fall through to the default
	// `redis://localhost:6379`.
	//
	// First env var wins per BindEnv; the long form stays supported
	// so a strictly-namespaced deployment is still valid.
	for key, envs := range envAliases {
		// BindEnv signature: name, env vars... — any of these resolve to key.
		if err := v.BindEnv(append([]string{key}, envs...)...); err != nil {
			return Config{}, fmt.Errorf("config: bind %s: %w", key, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: unmarshal: %w", err)
	}
	return cfg, nil
}

// envAliases maps config keys to the env var names we accept, in
// precedence order (first non-empty wins). Keep in sync with the
// README's configuration table.
var envAliases = map[string][]string{
	"cache.redis_url":     {"CSA_CACHE_REDIS_URL", "CSA_REDIS_URL"},
	"outbound.rate_limit": {"CSA_OUTBOUND_RATE_LIMIT"},
	"scraper.base_url":    {"CSA_SCRAPER_BASE_URL"},
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("cache.backend", "memory")
	v.SetDefault("cache.redis_url", "")
	v.SetDefault("scraper.base_url", "")
	v.SetDefault("outbound.rate_limit", 3.0)
}
