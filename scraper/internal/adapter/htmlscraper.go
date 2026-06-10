package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/gocolly/colly/v2"
)

type HTMLScraperConfig struct {
	Name          string
	SourceURL     string
	SeriesURL     string
	SeriesSel     string
	SeriesTitle   string
	SeriesLink    string
	ChapterSel    string
	ChapterNum    string
	ChapterLink   string
	ChapterTitle  string
}

type HTMLScraperAdapter struct {
	config HTMLScraperConfig
	client *colly.Collector
	log    *slog.Logger
}

func NewHTMLScraperAdapter(config HTMLScraperConfig) *HTMLScraperAdapter {
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		Parallelism: 1,
		Delay:       1 * time.Second,
	})

	return &HTMLScraperAdapter{
		config: config,
		client: c,
		log:    slog.Default().With("adapter", config.Name),
	}
}

func (h *HTMLScraperAdapter) Name() string {
	return h.config.Name
}

func (h *HTMLScraperAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	h.log.Warn("HTML scraper FetchLatest not implemented")
	return nil, nil
}

func (h *HTMLScraperAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	pageURL := fmt.Sprintf(h.config.SourceURL+"/%s", seriesID)
	var chapters []model.Chapter

	h.client.OnHTML(h.config.ChapterSel, func(e *colly.HTMLElement) {
		num := e.ChildText(h.config.ChapterNum)
		link := e.ChildAttr(h.config.ChapterLink, "href")
		title := e.ChildText(h.config.ChapterTitle)

		chapterNum := 0.0
		fmt.Sscanf(num, "%f", &chapterNum)

		chapters = append(chapters, model.Chapter{
			Number: chapterNum,
			Title:  title,
			URL:    h.config.SourceURL + link,
			IsNew:  true,
		})
	})

	if err := h.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("%s: visit: %w", h.config.Name, err)
	}

	h.client.Wait()
	return chapters, nil
}
