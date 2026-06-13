package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/aeswibon/manga-cdc/scraper/internal/validate"
	"github.com/aeswibon/manga-cdc/scraper/internal/watchlist"
)

const latestChapterJumpWindow = 40.0

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: discord-test-seed <mangaplus-title-id>")
		os.Exit(2)
	}

	titleID := os.Args[1]
	sourceID := watchlist.NamespacedSourceID("mangaplus", titleID)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()
	mp := adapter.NewMangaPlusAdapter()
	seriesMeta, err := mp.FetchSeries(ctx, titleID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch series: %v\n", err)
		os.Exit(1)
	}
	chapters, err := mp.FetchChapters(ctx, titleID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch chapters: %v\n", err)
		os.Exit(1)
	}
	if len(chapters) < 2 {
		fmt.Fprintln(os.Stderr, "need at least two chapters to simulate a new release")
		os.Exit(1)
	}

	sort.Slice(chapters, func(i, j int) bool { return chapters[i].Number < chapters[j].Number })
	latest := chapters[len(chapters)-1]
	seedChapters := chapters[:len(chapters)-1]
	seedLatestChapter := latest.Number - latestChapterJumpWindow

	database, err := db.New(ctx, dbURL, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	series := validate.NormalizeSeries(model.Series{
		SourceID:  sourceID,
		Title:     seriesMeta.Title,
		Author:    seriesMeta.Author,
		CoverURL:  seriesMeta.CoverURL,
		Description: seriesMeta.Description,
		SourceURL: seriesMeta.SourceURL,
		Status:    "ONGOING",
		IsActive:  true,
	})
	seriesResult := validate.Series(series, validate.Insert)
	if !seriesResult.OK {
		fmt.Fprintf(os.Stderr, "series validation failed: %+v\n", seriesResult.Issues)
		os.Exit(1)
	}

	seriesID, err := database.UpsertSeries(ctx, series)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upsert series: %v\n", err)
		os.Exit(1)
	}

	if err := database.DeleteChaptersForSeries(ctx, seriesID); err != nil {
		fmt.Fprintf(os.Stderr, "clear chapters: %v\n", err)
		os.Exit(1)
	}

	inserted, err := database.BulkInsertChapters(ctx, seriesID, seedChapters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed chapters: %v\n", err)
		os.Exit(1)
	}

	series.ID = seriesID
	series.LatestChapter = seedLatestChapter
	if err := database.UpdateSeries(ctx, series); err != nil {
		fmt.Fprintf(os.Stderr, "update latest_chapter: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("series_id=%s\n", seriesID)
	fmt.Printf("source_id=%s\n", sourceID)
	fmt.Printf("title=%s\n", series.Title)
	fmt.Printf("seeded_chapters=%d\n", len(inserted))
	fmt.Printf("withheld_chapter=%.0f %q\n", latest.Number, latest.Title)
	fmt.Printf("latest_chapter=%.0f\n", seedLatestChapter)
}
