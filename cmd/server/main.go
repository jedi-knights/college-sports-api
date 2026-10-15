// Package main is the college-sports-api entry point.
//
// Build with:
//
//	go build -o college-sports-api ./cmd/server
//
// Run with:
//
//	./college-sports-api serve --config ./config/config.example.yaml
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jedi-knights/go-logging/pkg/logging"
	"github.com/spf13/cobra"

	"github.com/jedi-knights/college-sports-api/internal/adapters/inbound/http"
	"github.com/jedi-knights/college-sports-api/internal/adapters/outbound/cache"
	"github.com/jedi-knights/college-sports-api/internal/adapters/outbound/scraper"
	"github.com/jedi-knights/college-sports-api/internal/config"
	"github.com/jedi-knights/college-sports-api/internal/container"
	"github.com/jedi-knights/college-sports-api/internal/domain"
	"github.com/jedi-knights/college-sports-api/internal/ports"
	"github.com/jedi-knights/college-sports-api/internal/sports/soccer"

	ghttp "github.com/jedi-knights/go-platform/httpserver"
)

// Version metadata set by ldflags at build time. See deploy/Dockerfile.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "college-sports-api",
		Short: "NCAA college sports data API",
	}
	cmd.AddCommand(newServeCmd(), newVersionCmd(), newHealthcheckCmd())
	return cmd
}

func newServeCmd() *cobra.Command {
	var cfgPath string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd.Context(), cfgPath)
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", defaultConfigPath(), "path to config.yaml")
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build metadata",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("college-sports-api %s (commit %s, built %s)\n", Version, Commit, BuildTime)
		},
	}
}

func newHealthcheckCmd() *cobra.Command {
	// Used by docker-compose healthcheck. Exits 0 if the server
	// is up and /readyz responds. Avoid shelling out to curl inside
	// the distroless image.
	return &cobra.Command{
		Use:    "healthcheck",
		Short:  "Hit /readyz on localhost and exit nonzero on failure",
		Hidden: true,
		RunE:   runHealthcheck,
	}
}

// defaultConfigPath follows the same convention as api-gateway:
// baked-in path first, then repo-local fallback for `task run`.
func defaultConfigPath() string {
	for _, p := range []string{
		"/etc/college-sports-api/config.yaml",
		"config/config.example.yaml",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func runServe(ctx context.Context, cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	logger := logging.New(logging.Config{
		Level:       cfg.Log.Level,
		Format:      cfg.Log.Format,
		ServiceName: "college-sports-api",
	})
	logger.Info("college-sports-api starting",
		"version", Version, "commit", Commit, "config", cfgPath,
		"cache_backend", cfg.Cache.Backend)

	cacheImpl, err := buildCache(cfg)
	if err != nil {
		return err
	}
	defer closeCache(cacheImpl)

	registry := buildRegistry(cfg)

	handler := http.NewRouter(http.Deps{
		Logger:   logger,
		Registry: registry,
		Cache:    cacheImpl,
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := ghttp.New(addr, handler)
	logger.Info("http server listening", "addr", addr)
	return server.Run(ctx)
}

func buildCache(cfg config.Config) (ports.Cache, error) {
	switch cfg.Cache.Backend {
	case "redis":
		return cache.NewRedis(cfg.Cache.RedisURL)
	case "memory", "":
		return cache.NewMemory(0), nil
	default:
		return nil, fmt.Errorf("config: unknown cache backend %q", cfg.Cache.Backend)
	}
}

func closeCache(c ports.Cache) {
	type closer interface {
		Close() error
	}
	if cc, ok := c.(closer); ok {
		_ = cc.Close()
	}
}

func buildRegistry(cfg config.Config) *container.Registry {
	ncaa := scraper.NewNCAAScoreboard(cfg.Scraper.BaseURL, nil)
	reg := container.NewRegistry()
	reg.RegisterSport(soccer.Men{}).
		RegisterSport(soccer.Women{}).
		RegisterScoreboardFetcher(domain.SportSoccerMen, ncaa).
		RegisterScoreboardFetcher(domain.SportSoccerWomen, ncaa)
	return reg
}
