package diff

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

type Engine struct {
	db  *db.DB
	log *slog.Logger
}

func New(database *db.DB, log *slog.Logger) *Engine {
	return &Engine{
		db:  database,
		log: log,
	}
}

type Result struct {
	NewChapters int
	SeriesID    string
	SeriesTitle string
	Chapters    []model.Chapter
}

func (e *Engine) ProcessSource(ctx context.Context, source adapter.SourceAdapter) ([]Result, error) {
	seriesList, err := source.FetchLatest(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch latest from %s: %w", source.Name(), err)
	}

	var results []Result

	for _, series := range seriesList {
		existingID, err := e.db.UpsertSeries(ctx, series)
		if err != nil {
			e.log.Error("failed to upsert series", "source", source.Name(), "title", series.Title, "error", err)
			continue
		}

		series.ID = existingID

		chapters, err := source.FetchChapters(ctx, series.SourceID)
		if err != nil {
			e.log.Error("failed to fetch chapters", "source", source.Name(), "series", series.Title, "error", err)
			continue
		}

		var newChapters []model.Chapter
		for _, ch := range chapters {
			id, err := e.db.InsertChapter(ctx, series.ID, ch)
			if err != nil {
				e.log.Error("failed to insert chapter", "source", source.Name(), "chapter", ch.Number, "error", err)
				continue
			}
			if id != "" {
				ch.ID = id
				ch.SeriesID = series.ID
				newChapters = append(newChapters, ch)
			}
		}

		if len(newChapters) > 0 {
			results = append(results, Result{
				NewChapters: len(newChapters),
				SeriesID:    series.ID,
				SeriesTitle: series.Title,
				Chapters:    newChapters,
			})
			e.log.Info("new chapters detected",
				"source", source.Name(),
				"series", series.Title,
				"count", len(newChapters))
		}
	}

	return results, nil
}
