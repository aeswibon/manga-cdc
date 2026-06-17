package diff

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/aeswibon/manga-cdc/scraper/internal/watchlist"
)

type stubAdapter struct {
	name     string
	chapters []model.Chapter
	err      error
}

func (s stubAdapter) Name() string { return s.name }

func (s stubAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	return nil, nil
}

func (s stubAdapter) FetchChapters(ctx context.Context, sourceID string) ([]model.Chapter, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.chapters, nil
}

func (s stubAdapter) FetchPages(ctx context.Context, chapterUrl string) ([]string, error) {
	return nil, nil
}

func TestMaybeStatusAlert(t *testing.T) {
	alert := maybeStatusAlert(model.Series{
		ID:     "id-1",
		Title:  "Test",
		Status: "HIATUS",
	}, "ONGOING")
	if alert == nil || alert.AlertType != "status_change" {
		t.Fatalf("expected hiatus alert, got %#v", alert)
	}

	if maybeStatusAlert(model.Series{Status: "ONGOING"}, "ONGOING") != nil {
		t.Fatal("expected nil when status unchanged")
	}
}

func TestFetchChaptersWithFallback_usesFallbackOnPrimaryError(t *testing.T) {
	engine := New(nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil)
	primary := stubAdapter{name: "mangadex", err: errors.New("blocked")}
	fallback := stubAdapter{
		name:     "mangafire",
		chapters: []model.Chapter{{Number: 1, URL: "https://example.com/1"}},
	}
	registry := map[string]adapter.SourceAdapter{
		"mangadex":  primary,
		"mangafire": fallback,
	}
	raw, _ := json.Marshal([]watchlist.FallbackSource{{
		Source: "mangafire", SourceID: "one-piece",
	}})
	series := model.Series{
		Title:           "One Piece",
		FallbackSources: raw,
	}

	chapters, used, err := engine.fetchChaptersWithFallback(context.Background(), registry, primary, "abc", series)
	if err != nil {
		t.Fatal(err)
	}
	if !used || len(chapters) != 1 {
		t.Fatalf("expected fallback chapters, got used=%v chapters=%v", used, chapters)
	}
}
