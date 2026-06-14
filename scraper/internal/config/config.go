package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/netguard"
)

type Config struct {
	DatabaseURL              string
	DBMaxConns               int
	ScrapeInterval           time.Duration
	ScrapeDelay              time.Duration
	WatchlistURL             string
	WatchlistPath            string
	WatchlistSyncInterval    time.Duration
	LogLevel                 string
	KafkaBrokers             string
	KafkaTopic               string
	KafkaUsername            string
	KafkaPassword            string
	QStashToken              string
	QStashDestination        string
	AdminDiscordWebhookURL   string
	DBEncryptionKey          string
	FlareSolverrURL          string
	ZeroResultAlertThreshold int
	RejectRateAlertThreshold float64
	RejectRateMinSample      int
	RunOnce                  bool
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func Load() (*Config, error) {
	interval := 5 * time.Minute
	if v := os.Getenv("SCRAPE_INTERVAL_SECONDS"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid SCRAPE_INTERVAL_SECONDS: %w", err)
		}
		interval = time.Duration(secs) * time.Second
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://mangacdc:mangacdc@localhost:5432/mangacdc?sslmode=disable"
	}

	dbMaxConns := 10
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err == nil && parsed > 0 {
			dbMaxConns = parsed
		}
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	adminWebhook := os.Getenv("ADMIN_DISCORD_WEBHOOK_URL")
	if adminWebhook == "" {
		adminWebhook = os.Getenv("DISCORD_WEBHOOK_URL")
	}

	threshold := 3
	if v := os.Getenv("ZERO_RESULT_ALERT_THRESHOLD"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ZERO_RESULT_ALERT_THRESHOLD: %w", err)
		}
		threshold = parsed
	}

	rejectRateThreshold := 0.5
	if v := os.Getenv("REJECT_RATE_ALERT_THRESHOLD"); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid REJECT_RATE_ALERT_THRESHOLD: %w", err)
		}
		rejectRateThreshold = parsed
	}

	rejectRateMinSample := 5
	if v := os.Getenv("REJECT_RATE_MIN_SAMPLE"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid REJECT_RATE_MIN_SAMPLE: %w", err)
		}
		rejectRateMinSample = parsed
	}

	scrapeDelay := 500 * time.Millisecond
	if v := os.Getenv("SCRAPE_DELAY_MS"); v != "" {
		ms, err := strconv.Atoi(v)
		if err == nil && ms >= 0 {
			scrapeDelay = time.Duration(ms) * time.Millisecond
		}
	}

	runOnce := false
	if v := os.Getenv("RUN_ONCE"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			runOnce = parsed
		}
	}

	watchlistSyncInterval := 24 * time.Hour
	if v := os.Getenv("WATCHLIST_SYNC_INTERVAL"); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid WATCHLIST_SYNC_INTERVAL: %w", err)
		}
		watchlistSyncInterval = parsed
	}

	watchlistPath := os.Getenv("WATCHLIST_PATH")
	if watchlistPath == "" {
		watchlistPath = "data/watchlist.yaml"
	}

	flareSolverrURL := os.Getenv("FLARESOLVERR_URL")
	if err := netguard.ValidateFlareSolverrURL(flareSolverrURL); err != nil {
		return nil, err
	}

	return &Config{
		DatabaseURL:              dbURL,
		DBMaxConns:               dbMaxConns,
		ScrapeInterval:           interval,
		ScrapeDelay:              scrapeDelay,
		WatchlistURL:             os.Getenv("WATCHLIST_URL"),
		WatchlistPath:            watchlistPath,
		WatchlistSyncInterval:    watchlistSyncInterval,
		LogLevel:                 logLevel,
		KafkaBrokers:             os.Getenv("KAFKA_BROKERS"),
		KafkaTopic:               getEnv("KAFKA_TOPIC", "mangacdc.public.chapters"),
		KafkaUsername:            os.Getenv("KAFKA_USERNAME"),
		KafkaPassword:            os.Getenv("KAFKA_PASSWORD"),
		QStashToken:              os.Getenv("QSTASH_TOKEN"),
		QStashDestination:        os.Getenv("QSTASH_DESTINATION_URL"),
		AdminDiscordWebhookURL:   adminWebhook,
		DBEncryptionKey:          os.Getenv("DB_ENCRYPTION_KEY"),
		FlareSolverrURL:          flareSolverrURL,
		ZeroResultAlertThreshold: threshold,
		RejectRateAlertThreshold: rejectRateThreshold,
		RejectRateMinSample:      rejectRateMinSample,
		RunOnce:                  runOnce,
	}, nil
}
