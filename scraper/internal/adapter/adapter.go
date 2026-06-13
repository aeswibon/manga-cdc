package adapter

import (
	"context"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

type SourceAdapter interface {
	Name() string
	FetchLatest(ctx context.Context) ([]model.Series, error)
	FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error)
	FetchPages(ctx context.Context, chapterUrl string) ([]string, error)
}

// SeriesMetadataFetcher is implemented by adapters that can load series details by source ID.
type SeriesMetadataFetcher interface {
	FetchSeries(ctx context.Context, seriesID string) (model.Series, error)
}

type SeriesUpdate struct {
	Series    model.Series
	Chapters  []model.Chapter
	FetchedAt time.Time
}
