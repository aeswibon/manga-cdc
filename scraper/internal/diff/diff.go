package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/aeswibon/manga-cdc/scraper/internal/validate"
	"github.com/aeswibon/manga-cdc/scraper/internal/watchlist"
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
	Results          []Result
	SeriesFetched    int
	SeriesAccepted   int
	SeriesRejected   int
	ChaptersRejected int
}

func (e *Engine) SyncWatchlist(ctx context.Context, entries []watchlist.Entry) (added int, rejected int, removed int, err error) {
	keepSourceIDs := make([]string, 0, len(entries))

	for _, entry := range entries {
		namespacedID := watchlist.NamespacedSourceID(entry.Source, entry.SourceID)
		keepSourceIDs = append(keepSourceIDs, namespacedID)

		existing, err := e.db.GetSeriesBySourceID(ctx, namespacedID)
		if err != nil {
			return added, rejected, removed, fmt.Errorf("check series %s: %w", namespacedID, err)
		}
		if existing != nil {
			continue
		}

		series := validate.NormalizeSeries(model.Series{
			SourceID:  namespacedID,
			Title:     entry.Title,
			SourceURL: entry.SourceURL,
			CoverURL:  entry.CoverURL,
			Status:    entry.Status,
			IsActive:  true,
		})

		seriesResult := validate.Series(series, validate.Insert)
		if !seriesResult.OK {
			rejected++
			validate.RecordReject(entry.Source, "series", seriesResult.Issues)
			e.quarantineReject(ctx, entry.Source, "series", series, seriesResult.Issues)
			e.log.Warn("rejected watchlist series",
				"source", entry.Source,
				"title", entry.Title,
				"issues", seriesResult.Issues)
			continue
		}

		validate.RecordAccept(entry.Source, "series")
		if _, err := e.db.UpsertSeries(ctx, series); err != nil {
			return added, rejected, removed, fmt.Errorf("upsert watchlist series %s: %w", namespacedID, err)
		}
		added++
		e.log.Info("watchlist series added", "source", entry.Source, "title", entry.Title)
	}

	deleted, err := e.db.DeleteSeriesExceptSourceIDs(ctx, keepSourceIDs)
	if err != nil {
		return added, rejected, removed, err
	}
	removed = int(deleted)
	if removed > 0 {
		e.log.Info("removed series not in watchlist", "count", removed)
	}

	return added, rejected, removed, nil
}

func (e *Engine) ProcessActiveSeries(ctx context.Context, source adapter.SourceAdapter, seriesList []model.Series) (SourceRun, error) {
	var results []Result
	run := SourceRun{SeriesFetched: len(seriesList)}

	for i, series := range seriesList {
		if i > 0 && e.seriesDelay > 0 {
			select {
			case <-ctx.Done():
				return SourceRun{}, ctx.Err()
			case <-time.After(e.seriesDelay):
			}
		}

		run.SeriesAccepted++

		_, rawID, err := watchlist.ParseRawSourceID(series.SourceID)
		if err != nil {
			e.log.Error("invalid namespaced source_id", "source_id", series.SourceID, "error", err)
			continue
		}

		if fetcher, ok := source.(adapter.SeriesMetadataFetcher); ok {
			meta, metaErr := fetcher.FetchSeries(ctx, rawID)
			if metaErr != nil {
				e.log.Warn("failed to fetch series metadata",
					"source", source.Name(),
					"series", series.Title,
					"error", metaErr)
			} else {
				series = validate.MergeSeries(series, meta)
			}
		}

		chapters, err := source.FetchChapters(ctx, rawID)
		if err != nil {
			e.log.Error("failed to fetch chapters", "source", source.Name(), "series", series.Title, "error", err)
			continue
		}

		chapterOpts := validate.ChapterOptions{LatestChapter: series.LatestChapter}
		goodChapters, rejectedChapters := validate.FilterChapters(chapters, chapterOpts)
		for _, rejected := range rejectedChapters {
			run.ChaptersRejected++
			validate.RecordReject(source.Name(), "chapter", rejected.Issues)
			e.quarantineReject(ctx, source.Name(), "chapter", rejected.Chapter, rejected.Issues)
			e.log.Warn("rejected chapter",
				"source", source.Name(),
				"series", series.Title,
				"chapter_num", rejected.Chapter.Number,
				"issues", rejected.Issues)
		}
		for range goodChapters {
			validate.RecordAccept(source.Name(), "chapter")
		}

		newChapters, err := e.db.BulkInsertChapters(ctx, series.ID, goodChapters)
		if err != nil {
			e.log.Error("failed to bulk insert chapters", "source", source.Name(), "series", series.Title, "error", err)
			continue
		}

		for _, ch := range goodChapters {
			if ch.Number > series.LatestChapter {
				series.LatestChapter = ch.Number
			}
		}

		if err := e.db.UpdateSeries(ctx, series); err != nil {
			e.log.Error("failed to update series last_checked", "source", source.Name(), "series", series.Title, "error", err)
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

	run.Results = results
	return run, nil
}

func (e *Engine) quarantineReject(ctx context.Context, source, entityType string, payload any, issues []validate.Issue) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		e.log.Error("failed to marshal quarantine payload", "source", source, "entity", entityType, "error", err)
		return
	}
	reasonsJSON, err := json.Marshal(issues)
	if err != nil {
		e.log.Error("failed to marshal quarantine reasons", "source", source, "entity", entityType, "error", err)
		return
	}
	if err := e.db.InsertScrapedReject(ctx, source, entityType, payloadJSON, reasonsJSON); err != nil {
		e.log.Error("failed to quarantine rejected record", "source", source, "entity", entityType, "error", err)
	}
}
