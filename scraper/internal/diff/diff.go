package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
	"github.com/aeswibon/manga-cdc/scraper/internal/db"
	"github.com/aeswibon/manga-cdc/scraper/internal/metadata"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/aeswibon/manga-cdc/scraper/internal/validate"
	"github.com/aeswibon/manga-cdc/scraper/internal/watchlist"
)

type Engine struct {
	db          *db.DB
	log         *slog.Logger
	seriesDelay time.Duration
	resolver    *metadata.Resolver
}

func New(database *db.DB, log *slog.Logger, resolver *metadata.Resolver) *Engine {
	return &Engine{
		db:          database,
		log:         log,
		seriesDelay: 500 * time.Millisecond,
		resolver:    resolver,
	}
}

func NewWithDelay(database *db.DB, log *slog.Logger, resolver *metadata.Resolver, delay time.Duration) *Engine {
	return &Engine{
		db:          database,
		log:         log,
		seriesDelay: delay,
		resolver:    resolver,
	}
}

type Result struct {
	NewChapters int
	SeriesID    string
	SeriesTitle string
	Chapters    []model.Chapter
}

type SeriesAlert struct {
	SeriesID  string
	Title     string
	Status    string
	SourceURL string
	AlertType string
}

type SourceRun struct {
	Results          []Result
	SeriesAlerts     []SeriesAlert
	SeriesFetched    int
	SeriesAccepted   int
	SeriesRejected   int
	ChaptersRejected int
	FallbackUsed     int
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
		prefsJSON := entry.NotificationPrefsJSON()
		if existing != nil {
			if err := e.db.UpdateSeriesWatchlistMeta(ctx, namespacedID, prefsJSON, entry.FallbackSourcesJSON()); err != nil {
				return added, rejected, removed, fmt.Errorf("sync watchlist meta %s: %w", namespacedID, err)
			}
			continue
		}

		series := validate.NormalizeSeries(model.Series{
			SourceID:          namespacedID,
			Title:             entry.Title,
			SourceURL:         entry.SourceURL,
			CoverURL:          entry.CoverURL,
			Status:            entry.Status,
			IsActive:          true,
			NotificationPrefs: prefsJSON,
			FallbackSources:   entry.FallbackSourcesJSON(),
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

func (e *Engine) ProcessActiveSeries(ctx context.Context, registry map[string]adapter.SourceAdapter, primary adapter.SourceAdapter, seriesList []model.Series) (SourceRun, error) {
	var results []Result
	var alerts []SeriesAlert
	run := SourceRun{SeriesFetched: len(seriesList)}

	for i, series := range seriesList {
		if i > 0 && e.seriesDelay > 0 {
			select {
			case <-ctx.Done():
				return SourceRun{}, ctx.Err()
			case <-time.After(e.seriesDelay):
			}
		}

		_, rawID, err := watchlist.ParseRawSourceID(series.SourceID)
		if err != nil {
			e.log.Error("invalid namespaced source_id", "source_id", series.SourceID, "error", err)
			run.SeriesRejected++
			continue
		}

		previousStatus := series.Status

		if fetcher, ok := primary.(adapter.SeriesMetadataFetcher); ok {
			meta, metaErr := fetcher.FetchSeries(ctx, rawID)
			if metaErr != nil {
				e.log.Warn("failed to fetch series metadata",
					"source", primary.Name(),
					"series", series.Title,
					"error", metaErr)
			} else {
				series = validate.MergeSeries(series, validate.NormalizeSeries(meta))
			}
		}

		if series.AniListID == nil && e.resolver != nil {
			md, err := e.resolver.Resolve(ctx, series.Title, series.AltTitles)
			if err != nil {
				e.log.Warn("failed to resolve metadata", "series", series.Title, "error", err)
			} else if md != nil {
				series.AniListID = &md.AniListID
				series.MalID = md.MalID
				series.CanonicalTitle = md.CanonicalTitle
			}
		}

		series = validate.NormalizeSeries(series)
		seriesResult := validate.Series(series, validate.Update)
		if !seriesResult.OK {
			run.SeriesRejected++
			validate.RecordReject(primary.Name(), "series", seriesResult.Issues)
			e.quarantineReject(ctx, primary.Name(), "series", series, seriesResult.Issues)
			e.log.Warn("rejected series metadata",
				"source", primary.Name(),
				"series", series.Title,
				"issues", seriesResult.Issues)
			continue
		}
		validate.RecordAccept(primary.Name(), "series")
		run.SeriesAccepted++

		if statusAlert := maybeStatusAlert(series, previousStatus); statusAlert != nil {
			alerts = append(alerts, *statusAlert)
		}

		if err := e.db.UpdateSeries(ctx, series); err != nil {
			e.log.Error("failed to persist series metadata",
				"source", primary.Name(),
				"series", series.Title,
				"error", err)
			continue
		}

		chapters, usedFallback, err := e.fetchChaptersWithFallback(ctx, registry, primary, rawID, series)
		if err != nil {
			e.log.Error("failed to fetch chapters", "source", primary.Name(), "series", series.Title, "error", err)
			continue
		}
		if usedFallback {
			run.FallbackUsed++
		}

		chapterOpts := validate.ChapterOptions{LatestChapter: series.LatestChapter}
		goodChapters, rejectedChapters := validate.FilterChapters(chapters, chapterOpts)
		for _, rejected := range rejectedChapters {
			run.ChaptersRejected++
			validate.RecordReject(primary.Name(), "chapter", rejected.Issues)
			e.quarantineReject(ctx, primary.Name(), "chapter", rejected.Chapter, rejected.Issues)
			e.log.Warn("rejected chapter",
				"source", primary.Name(),
				"series", series.Title,
				"chapter_num", rejected.Chapter.Number,
				"issues", rejected.Issues)
		}
		for range goodChapters {
			validate.RecordAccept(primary.Name(), "chapter")
		}

		newChapters, err := e.db.BulkInsertChapters(ctx, series.ID, goodChapters)
		if err != nil {
			e.log.Error("failed to bulk insert chapters", "source", primary.Name(), "series", series.Title, "error", err)
			continue
		}

		for _, ch := range goodChapters {
			if ch.Number > series.LatestChapter {
				series.LatestChapter = ch.Number
			}
		}

		if err := e.db.UpdateSeries(ctx, series); err != nil {
			e.log.Error("failed to update series last_checked", "source", primary.Name(), "series", series.Title, "error", err)
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
				"source", primary.Name(),
				"series", series.Title,
				"count", len(newChapters))
		}
	}

	run.Results = results
	run.SeriesAlerts = alerts
	return run, nil
}

func maybeStatusAlert(series model.Series, previousStatus string) *SeriesAlert {
	if series.Status == "" || series.Status == previousStatus {
		return nil
	}
	if series.Status != "HIATUS" && series.Status != "COMPLETED" {
		return nil
	}
	return &SeriesAlert{
		SeriesID:  series.ID,
		Title:     series.Title,
		Status:    series.Status,
		SourceURL: series.SourceURL,
		AlertType: "status_change",
	}
}

func (e *Engine) fetchChaptersWithFallback(
	ctx context.Context,
	registry map[string]adapter.SourceAdapter,
	primary adapter.SourceAdapter,
	rawID string,
	series model.Series,
) ([]model.Chapter, bool, error) {
	chapters, err := primary.FetchChapters(ctx, rawID)
	if err == nil {
		return chapters, false, nil
	}

	e.log.Warn("primary chapter fetch failed, trying fallbacks",
		"source", primary.Name(),
		"series", series.Title,
		"error", err)

	fallbacks, err := parseFallbackSources(series.FallbackSources)
	if err != nil {
		return nil, false, fmt.Errorf("parse fallback_sources: %w", err)
	}

	var lastErr error
	for _, fb := range fallbacks {
		adapterInstance, ok := registry[fb.Source]
		if !ok {
			continue
		}
		chapters, fbErr := adapterInstance.FetchChapters(ctx, fb.SourceID)
		if fbErr == nil {
			e.log.Info("used fallback source for chapters",
				"primary", primary.Name(),
				"fallback", fb.Source,
				"series", series.Title)
			return chapters, true, nil
		}
		lastErr = fbErr
		e.log.Warn("fallback chapter fetch failed",
			"fallback", fb.Source,
			"series", series.Title,
			"error", fbErr)
	}

	if lastErr != nil {
		return nil, false, lastErr
	}
	return nil, false, err
}

func parseFallbackSources(raw json.RawMessage) ([]watchlist.FallbackSource, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var fallbacks []watchlist.FallbackSource
	if err := json.Unmarshal(raw, &fallbacks); err != nil {
		return nil, err
	}
	return fallbacks, nil
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
