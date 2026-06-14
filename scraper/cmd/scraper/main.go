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
	"github.com/aeswibon/manga-cdc/scraper/internal/alert"
	"github.com/aeswibon/manga-cdc/scraper/internal/archive"
	"github.com/aeswibon/manga-cdc/scraper/internal/config"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/diff"
	"github.com/aeswibon/manga-cdc/scraper/internal/health"
	"github.com/aeswibon/manga-cdc/scraper/internal/metadata"
	"github.com/aeswibon/manga-cdc/scraper/internal/migrate"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/aeswibon/manga-cdc/scraper/internal/qstash"
	"github.com/aeswibon/manga-cdc/scraper/internal/version"
	"github.com/aeswibon/manga-cdc/scraper/internal/watchlist"
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

	chaptersPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scraper_chapters_published_total",
		Help: "Total chapters published to message broker",
	})
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if err := migrate.Run(ctx, cfg.DatabaseURL); err != nil {
		log.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}

	database, err := db.New(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	resolver := metadata.NewResolver()
	engine := diff.NewWithDelay(database, log, resolver, cfg.ScrapeDelay)

	var qstashPublisher *qstash.Publisher
	if cfg.QStashToken != "" && cfg.QStashDestination != "" {
		qstashPublisher = qstash.NewPublisher(cfg.QStashToken, cfg.QStashDestination)
		log.Info("QStash publisher enabled", "destination", cfg.QStashDestination)
	}

	zeroMonitor := alert.New(log, cfg.AdminDiscordWebhookURL, alert.Config{
		ZeroResultThreshold:   cfg.ZeroResultAlertThreshold,
		RejectRateThreshold:   cfg.RejectRateAlertThreshold,
		RejectRateMinSample:   cfg.RejectRateMinSample,
	})
	if cfg.AdminDiscordWebhookURL != "" {
		log.Info("scraper alerts enabled",
			"zero_result_threshold", cfg.ZeroResultAlertThreshold,
			"reject_rate_threshold", cfg.RejectRateAlertThreshold,
			"reject_rate_min_sample", cfg.RejectRateMinSample)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	health.New(database, nil, false).Register(mux)
	metricsServer := &http.Server{Addr: ":2112", Handler: mux}
	go func() {
		log.Info("metrics server starting", "addr", ":2112")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics server error", "error", err)
		}
	}()

	sourceRegistry := map[string]adapter.SourceAdapter{
		"mangadex":   adapter.NewMangaDexAdapter(),
		"mangafire":  adapter.NewMangaFireAdapter(),
		"asurascans": adapter.NewAsuraScansAdapter(),
		"mangaplus":  adapter.NewMangaPlusAdapter(),
		"mangatown":  adapter.NewMangaTownAdapter(),
		"mangapill":  adapter.NewMangaPillAdapter(),
	}

	log.Info("scraper started",
		"version", version.Version,
		"sources", len(sourceRegistry),
		"scrape_interval", cfg.ScrapeInterval,
		"watchlist_sync_interval", cfg.WatchlistSyncInterval,
		"watchlist_url", cfg.WatchlistURL,
		"watchlist_path", cfg.WatchlistPath)

	archiver := archive.NewArchiver("/archives")
	log.Info("archiver enabled", "base_dir", "/archives")

	syncWatchlist(ctx, log, engine, cfg)

	scrapeActiveSeries(ctx, log, engine, zeroMonitor, database, sourceRegistry, qstashPublisher, archiver)

	if cfg.RunOnce {
		log.Info("run_once enabled, exiting scrape run")
		return
	}

	watchlistTicker := time.NewTicker(cfg.WatchlistSyncInterval)
	defer watchlistTicker.Stop()
	scrapeTicker := time.NewTicker(cfg.ScrapeInterval)
	defer scrapeTicker.Stop()

	for {
		select {
		case <-sigCh:
			log.Info("shutting down...")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			metricsServer.Shutdown(shutdownCtx)
			cancel()
			return
		case <-watchlistTicker.C:
			syncWatchlist(ctx, log, engine, cfg)
		case <-scrapeTicker.C:
			scrapeActiveSeries(ctx, log, engine, zeroMonitor, database, sourceRegistry, qstashPublisher, archiver)
		}
	}
}

func loadWatchlistEntries(ctx context.Context, cfg *config.Config) ([]watchlist.Entry, error) {
	if cfg.WatchlistURL != "" {
		return watchlist.LoadFromURL(ctx, cfg.WatchlistURL)
	}
	return watchlist.LoadFromFile(cfg.WatchlistPath)
}

func syncWatchlist(ctx context.Context, log *slog.Logger, engine *diff.Engine, cfg *config.Config) {
	entries, err := loadWatchlistEntries(ctx, cfg)
	if err != nil {
		log.Error("watchlist sync failed", "error", err)
		return
	}

	added, rejected, removed, err := engine.SyncWatchlist(ctx, entries)
	if err != nil {
		log.Error("watchlist sync failed", "error", err)
		return
	}
	log.Info("watchlist sync complete", "entries", len(entries), "added", added, "rejected", rejected, "removed", removed)
}

func scrapeActiveSeries(
	ctx context.Context,
	log *slog.Logger,
	engine *diff.Engine,
	zeroMonitor *alert.Monitor,
	database *db.DB,
	sourceRegistry map[string]adapter.SourceAdapter,
	qstashPublisher *qstash.Publisher,
	archiver *archive.Archiver,
) {
	activeSeries, err := database.GetActiveSeries(ctx)
	if err != nil {
		log.Error("failed to load active series", "error", err)
		return
	}

	bySource := groupActiveSeriesBySource(activeSeries)

	for name, source := range sourceRegistry {
		seriesForSource := bySource[name]
		scrapeSource(ctx, log, engine, zeroMonitor, source, seriesForSource, qstashPublisher, archiver)
	}
}

func groupActiveSeriesBySource(series []model.Series) map[string][]model.Series {
	grouped := make(map[string][]model.Series)
	for _, s := range series {
		source, _, err := watchlist.ParseRawSourceID(s.SourceID)
		if err != nil {
			continue
		}
		grouped[source] = append(grouped[source], s)
	}
	return grouped
}

func scrapeSource(
	ctx context.Context,
	log *slog.Logger,
	engine *diff.Engine,
	zeroMonitor *alert.Monitor,
	source adapter.SourceAdapter,
	activeSeries []model.Series,
	qstashPublisher *qstash.Publisher,
	archiver *archive.Archiver,
) {
	start := time.Now()
	run, err := engine.ProcessActiveSeries(ctx, source, activeSeries)
	duration := time.Since(start).Seconds()

	if err != nil {
		scrapeErrors.WithLabelValues(source.Name(), "process").Inc()
		log.Error("source scrape failed", "name", source.Name(), "error", err)
		return
	}

	scrapeDuration.WithLabelValues(source.Name()).Observe(duration)
	zeroMonitor.RecordScrape(ctx, source.Name(), run.SeriesFetched)
	zeroMonitor.RecordValidation(ctx, source.Name(), run.SeriesFetched, run.SeriesRejected)

	for _, r := range run.Results {
		chaptersDetected.Add(float64(r.NewChapters))
		log.Info("source update",
			"source", source.Name(),
			"series", r.SeriesTitle,
			"new_chapters", r.NewChapters)

		if qstashPublisher != nil {
			for _, ch := range r.Chapters {
				if err := qstashPublisher.PublishChapterEvent(ctx, ch); err != nil {
					log.Error("failed to publish via QStash",
						"source", source.Name(),
						"series", r.SeriesTitle,
						"chapter", ch.Number,
						"error", err)
					continue
				}
				chaptersPublished.Inc()
			}
		}

		if archiver != nil {
			for _, ch := range r.Chapters {
				pages, err := source.FetchPages(ctx, ch.URL)
				if err != nil {
					log.Error("failed to fetch pages for archive", "source", source.Name(), "chapter", ch.Number, "error", err)
					continue
				}
				if err := archiver.ArchiveChapter(ctx, model.Series{Title: r.SeriesTitle}, ch, pages); err != nil {
					log.Error("failed to archive chapter", "source", source.Name(), "chapter", ch.Number, "error", err)
				} else {
					log.Info("chapter archived", "source", source.Name(), "series", r.SeriesTitle, "chapter", ch.Number)
				}
			}
		}
	}

	if len(activeSeries) == 0 {
		log.Debug("no active series for source", "source", source.Name())
	} else if len(run.Results) == 0 {
		log.Debug("no new chapters", "source", source.Name(), "series_polled", len(activeSeries))
	}
}
