package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
	"github.com/aeswibon/manga-cdc/scraper/internal/config"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/diff"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	scrapeDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "scraper_duration_seconds",
		Help:    "Duration of scrape cycles per source",
		Buckets: prometheus.DefBuckets,
	}, []string{"source"})

	scrapeErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "scraper_errors_total",
		Help: "Total scrape errors per source",
	}, []string{"source", "type"})

	chaptersDetected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scraper_chapters_detected_total",
		Help: "Total new chapters detected",
	})
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

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	metricsServer := &http.Server{Addr: ":2112", Handler: mux}
	go func() {
		log.Info("metrics server starting", "addr", ":2112")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics server error", "error", err)
		}
	}()

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	engine := diff.New(database, log)

	sources := []adapter.SourceAdapter{
		adapter.NewMangaDexAdapter(),
	}

	log.Info("scraper started", "sources", len(sources), "interval", cfg.ScrapeInterval)

	ticker := time.NewTicker(cfg.ScrapeInterval)
	defer ticker.Stop()

	for _, source := range sources {
		scrapeSource(ctx, log, engine, source)
	}

	for {
		select {
		case <-sigCh:
			log.Info("shutting down...")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			metricsServer.Shutdown(shutdownCtx)
			cancel()
			return
		case <-ticker.C:
		}

		for _, source := range sources {
			scrapeSource(ctx, log, engine, source)
		}
	}
}

func scrapeSource(ctx context.Context, log *slog.Logger, engine *diff.Engine, source adapter.SourceAdapter) {
	start := time.Now()
	results, err := engine.ProcessSource(ctx, source)
	duration := time.Since(start).Seconds()

	if err != nil {
		scrapeErrors.WithLabelValues(source.Name(), "process").Inc()
		log.Error("source scrape failed", "name", source.Name(), "error", err)
		return
	}

	scrapeDuration.WithLabelValues(source.Name()).Observe(duration)

	for _, r := range results {
		chaptersDetected.Add(float64(r.NewChapters))
		log.Info("source update",
			"source", source.Name(),
			"series", r.SeriesTitle,
			"new_chapters", r.NewChapters)
	}
}
