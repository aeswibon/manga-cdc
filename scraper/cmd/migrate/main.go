package main

import (
	"context"
	"log"
	"os"

	"github.com/aeswibon/manga-cdc/scraper/internal/migrate"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if err := migrate.Run(context.Background(), dsn); err != nil {
		log.Fatal(err)
	}
}
