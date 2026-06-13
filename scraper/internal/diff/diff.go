package diff

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

type Engine struct {
	db          *db.DB
	log         *slog.Logger
	seriesDelay time.Duration
}

func New(database *db.DB, log *slog.Logger) *Engine {
	return &Engine{
		db:          database,
		log:         log,
		seriesDelay: 500 * time.Millisecond,
	}
}

func NewWithDelay(database *db.DB, log *slog.Logger, delay time.Duration) *Engine {
	return &Engine{
		db:          database,
		log:         log,
		seriesDelay: delay,
	}
}

type Result struct {
	NewChapters int
	SeriesID    string
	SeriesTitle string
	Chapters    []model.Chapter
}

type SourceRun struct {
	Results       []Result
	SeriesFetched int
}

func (e *Engine) ProcessSource(ctx context.Context, source adapter.SourceAdapter) (SourceRun, error) {
	seriesList, err := source.FetchLatest(ctx)
	if err != nil {
		return SourceRun{}, fmt.Errorf("fetch latest from %s: %w", source.Name(), err)
	}

	var results []Result

	for i, series := range seriesList {
		if i > 0 && e.seriesDelay > 0 {
			select {
			case <-ctx.Done():
				return SourceRun{}, ctx.Err()
			case <-time.After(e.seriesDelay):
			}
		}

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

		newChapters, err := e.db.BulkInsertChapters(ctx, series.ID, chapters)
		if err != nil {
			e.log.Error("failed to bulk insert chapters", "source", source.Name(), "series", series.Title, "error", err)
			continue
		}

		for i := range newChapters {
			newChapters[i].SeriesTitle = series.Title
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

	return SourceRun{
		Results:       results,
		SeriesFetched: len(seriesList),
	}, nil
}
