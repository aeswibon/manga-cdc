package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
	"github.com/aeswibon/manga-cdc/scraper/internal/config"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/diff"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.LogLevel == "debug" {
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	engine := diff.New(database)

	sources := []adapter.SourceAdapter{
		adapter.NewMangaDexAdapter(),
	}

	log.Info("scraper started", "sources", len(sources), "interval", cfg.ScrapeInterval)

	ticker := time.NewTicker(cfg.ScrapeInterval)
	defer ticker.Stop()

	for {
		for _, source := range sources {
			log.Debug("scraping source", "name", source.Name())
			results, err := engine.ProcessSource(ctx, source)
			if err != nil {
				log.Error("source scrape failed", "name", source.Name(), "error", err)
				continue
			}
			for _, r := range results {
				log.Info("source update",
					"source", source.Name(),
					"series", r.SeriesTitle,
					"new_chapters", r.NewChapters)
			}
		}

		select {
		case <-sigCh:
			fmt.Println()
			log.Info("shutting down...")
			return
		case <-ticker.C:
		}
	}
}
