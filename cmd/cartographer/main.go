// Package main is the entrypoint for the cartographer MCP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/sgaunet/cartographer-mcp/internal/cache"
	"github.com/sgaunet/cartographer-mcp/internal/config"
	"github.com/sgaunet/cartographer-mcp/internal/crawler"
	mcpserver "github.com/sgaunet/cartographer-mcp/internal/mcp"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	refresh := flag.Bool("refresh", false, "Force a full cache refresh on startup before serving")
	flag.Parse()

	if *showVersion {
		fmt.Println("cartographer-mcp", version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	slog.Info("starting cartographer-mcp",
		"version", version,
		"gitlab_uri", cfg.GitLabURI,
		"groups", cfg.Groups,
		"cache_dir", cfg.CacheDir)

	store := cache.NewStore(cfg.CacheDir)
	if err := store.Load(); err != nil {
		slog.Error("failed to load cache", "error", err)
		os.Exit(1)
	}

	c, err := crawler.New(cfg.GitLabToken, cfg.GitLabURI)
	if err != nil {
		slog.Error("failed to create crawler", "error", err)
		os.Exit(1)
	}

	refresher := cache.NewRefresher(store, c)

	if *refresh {
		if len(cfg.Groups) == 0 {
			fmt.Fprintln(os.Stderr, "--refresh requires CARTOGRAPHER_GROUPS to be configured")
			os.Exit(1)
		}
		slog.Info("refreshing cache on startup", "groups", cfg.Groups)
		result, err := refresher.Refresh(context.Background(), cfg.GitLabURI, cfg.Groups)
		if err != nil {
			slog.Error("startup refresh failed", "error", err)
			os.Exit(1)
		}
		slog.Info("startup refresh complete",
			"projects_discovered", result.ProjectsDiscovered,
			"projects_with_cartographer", result.ProjectsWithCartographer,
			"duration_ms", result.DurationMs,
			"diagnostics", len(result.Diagnostics))
	}

	srv := mcpserver.NewServer(cfg, store, refresher)
	if err := srv.ServeStdio(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
