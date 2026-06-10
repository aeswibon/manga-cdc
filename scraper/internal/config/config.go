package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL    string
	ScrapeInterval time.Duration
	LogLevel       string
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

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return &Config{
		DatabaseURL:    dbURL,
		ScrapeInterval: interval,
		LogLevel:       logLevel,
	}, nil
}
