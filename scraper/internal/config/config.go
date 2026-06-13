package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL              string
	DBMaxConns               int
	ScrapeInterval           time.Duration
	LogLevel                 string
	KafkaBrokers             string
	KafkaTopic               string
	KafkaUsername            string
	KafkaPassword            string
	QStashToken              string
	QStashDestination        string
	AdminDiscordWebhookURL   string
	ZeroResultAlertThreshold int
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

	runOnce := false
	if v := os.Getenv("RUN_ONCE"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			runOnce = parsed
		}
	}

	return &Config{
		DatabaseURL:              dbURL,
		DBMaxConns:               dbMaxConns,
		ScrapeInterval:           interval,
		LogLevel:                 logLevel,
		KafkaBrokers:             os.Getenv("KAFKA_BROKERS"),
		KafkaTopic:               getEnv("KAFKA_TOPIC", "mangacdc.public.chapters"),
		KafkaUsername:            os.Getenv("KAFKA_USERNAME"),
		KafkaPassword:            os.Getenv("KAFKA_PASSWORD"),
		QStashToken:              os.Getenv("QSTASH_TOKEN"),
		QStashDestination:        os.Getenv("QSTASH_DESTINATION_URL"),
		AdminDiscordWebhookURL:   adminWebhook,
		ZeroResultAlertThreshold: threshold,
		RunOnce:                  runOnce,
	}, nil
}
