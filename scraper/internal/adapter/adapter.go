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
}

type SeriesUpdate struct {
	Series    model.Series
	Chapters  []model.Chapter
	FetchedAt time.Time
}
