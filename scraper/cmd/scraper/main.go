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
	"github.com/aeswibon/manga-cdc/scraper/internal/config"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/diff"
	"github.com/aeswibon/manga-cdc/scraper/internal/health"
	"github.com/aeswibon/manga-cdc/scraper/internal/kafka"
	"github.com/aeswibon/manga-cdc/scraper/internal/migrate"
	"github.com/aeswibon/manga-cdc/scraper/internal/qstash"
	"github.com/aeswibon/manga-cdc/scraper/internal/version"
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

	if cfg.LogLevel == "debug" {
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if err := migrate.Run(ctx, cfg.DatabaseURL); err != nil {
		log.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	engine := diff.New(database, log)

	var kafkaProducer *kafka.Producer
	if cfg.KafkaBrokers != "" {
		kafkaProducer, err = kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaUsername, cfg.KafkaPassword)
		if err != nil {
			log.Error("failed to create Kafka producer", "error", err)
			os.Exit(1)
		}
		defer kafkaProducer.Close()
		log.Info("Kafka producer enabled", "brokers", cfg.KafkaBrokers, "topic", cfg.KafkaTopic)
	}

	var qstashPublisher *qstash.Publisher
	if cfg.QStashToken != "" && cfg.QStashDestination != "" {
		qstashPublisher = qstash.NewPublisher(cfg.QStashToken, cfg.QStashDestination)
		log.Info("QStash publisher enabled", "destination", cfg.QStashDestination)
	}

	zeroMonitor := alert.New(log, cfg.AdminDiscordWebhookURL, cfg.ZeroResultAlertThreshold)
	if cfg.AdminDiscordWebhookURL != "" {
		log.Info("zero-result alerts enabled", "threshold", cfg.ZeroResultAlertThreshold)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	health.New(database, kafkaProducer, cfg.KafkaBrokers != "").Register(mux)
	metricsServer := &http.Server{Addr: ":2112", Handler: mux}
	go func() {
		log.Info("metrics server starting", "addr", ":2112")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics server error", "error", err)
		}
	}()

	sources := []adapter.SourceAdapter{
		adapter.NewMangaDexAdapter(),
		adapter.NewMangaFireAdapter(),
		adapter.NewAsuraScansAdapter(),
		adapter.NewMangaPlusAdapter(),
		adapter.NewMangaTownAdapter(),
		adapter.NewMangaPillAdapter(),
	}

	log.Info("scraper started", "version", version.Version, "sources", len(sources), "interval", cfg.ScrapeInterval)

	ticker := time.NewTicker(cfg.ScrapeInterval)
	defer ticker.Stop()

	for _, source := range sources {
		scrapeSource(ctx, log, engine, zeroMonitor, source, kafkaProducer, qstashPublisher)
	}

	if cfg.RunOnce {
		log.Info("run_once enabled, exiting scrape run")
		return
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
			scrapeSource(ctx, log, engine, zeroMonitor, source, kafkaProducer, qstashPublisher)
		}
	}
}

func scrapeSource(
	ctx context.Context,
	log *slog.Logger,
	engine *diff.Engine,
	zeroMonitor *alert.Monitor,
	source adapter.SourceAdapter,
	kafkaProducer *kafka.Producer,
	qstashPublisher *qstash.Publisher,
) {
	start := time.Now()
	run, err := engine.ProcessSource(ctx, source)
	duration := time.Since(start).Seconds()

	if err != nil {
		scrapeErrors.WithLabelValues(source.Name(), "process").Inc()
		log.Error("source scrape failed", "name", source.Name(), "error", err)
		return
	}

	scrapeDuration.WithLabelValues(source.Name()).Observe(duration)
	zeroMonitor.RecordScrape(ctx, source.Name(), run.SeriesFetched)

	for _, r := range run.Results {
		chaptersDetected.Add(float64(r.NewChapters))
		log.Info("source update",
			"source", source.Name(),
			"series", r.SeriesTitle,
			"new_chapters", r.NewChapters)

		if kafkaProducer != nil {
			for _, ch := range r.Chapters {
				if err := kafkaProducer.PublishChapterEvent(ctx, ch); err != nil {
					log.Error("failed to publish chapter event",
						"source", source.Name(),
						"series", r.SeriesTitle,
						"chapter", ch.Number,
						"error", err)
					continue
				}
				chaptersPublished.Inc()
			}
		}

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
	}
}
