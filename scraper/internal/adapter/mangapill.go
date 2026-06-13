package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/gocolly/colly/v2"
)

const mangapillBase = "https://mangapill.com"

type MangaPillAdapter struct {
	log    *slog.Logger
	client *colly.Collector
}

func NewMangaPillAdapter() *MangaPillAdapter {
	return &MangaPillAdapter{
		log: slog.Default().With("adapter", "mangapill"),
	}
}

func (m *MangaPillAdapter) Name() string {
	return "mangapill"
}

func (m *MangaPillAdapter) newCollector() *colly.Collector {
	if m.client != nil {
		return m.client
	}
	c := colly.NewCollector(
		colly.AllowedDomains("mangapill.com"),
	)
	configureHTMLCollector(c)
	return c
}

func (m *MangaPillAdapter) SetCollector(c *colly.Collector) {
	m.client = c
}

func (m *MangaPillAdapter) FetchSeries(ctx context.Context, seriesID string) (model.Series, error) {
	parts := strings.SplitN(seriesID, "/", 2)
	id := parts[0]
	slug := ""
	if len(parts) > 1 {
		slug = parts[1]
	}
	pageURL := mangapillBase + "/manga/" + id + "/" + slug
	c := m.newCollector()
	meta, err := scrapeOpenGraph(c, pageURL)
	if err != nil {
		return model.Series{}, fmt.Errorf("mangapill: %w", err)
	}
	return model.Series{
		SourceID:    seriesID,
		Title:       meta.Title,
		Description: meta.Description,
		CoverURL:    meta.Image,
		SourceURL:   pageURL,
		Status:      "ONGOING",
	}, nil
}

func (m *MangaPillAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	pageURL := mangapillBase + "/"
	seen := make(map[string]bool)
	var series []model.Series
	var mu sync.Mutex

	c := m.newCollector()

	c.OnHTML("div.rounded", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		mangaLink := e.ChildAttr("a[href^='/manga/']", "href")
		if mangaLink == "" {
			return
		}

		parts := strings.Split(strings.Trim(mangaLink, "/"), "/")
		if len(parts) < 3 {
			return
		}
		seriesID := parts[len(parts)-2]
		slug := parts[len(parts)-1]

		if seen[seriesID] {
			return
		}
		seen[seriesID] = true

		title := strings.TrimSpace(e.ChildText("a[href^='/manga/'] div"))
		if title == "" {
			title = slug
		}

		cover := e.ChildAttr("img", "data-src")
		if cover == "" {
			cover = e.ChildAttr("img", "src")
		}

		series = append(series, model.Series{
			SourceID:  seriesID + "/" + slug,
			SourceURL: mangapillBase + mangaLink,
			Title:     title,
			CoverURL:  cover,
			Status:    "ONGOING",
			IsActive:  true,
		})
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangapill: visit %s: %w", pageURL, err)
	}
	c.Wait()

	return series, nil
}

func (m *MangaPillAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	parts := strings.SplitN(seriesID, "/", 2)
	id := parts[0]
	slug := ""
	if len(parts) > 1 {
		slug = parts[1]
	}

	pageURL := mangapillBase + "/manga/" + id + "/" + slug

	var chapters []model.Chapter
	var mu sync.Mutex

	c := m.newCollector()

	c.OnHTML("div#chapters a[href^='/chapters/']", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		text := strings.TrimSpace(e.Text)
		href := e.Attr("href")
		if text == "" || href == "" {
			return
		}

		cNum := extractMangaPillChapterNum(text, href)
		if math.IsNaN(cNum) {
			return
		}

		chapters = append(chapters, model.Chapter{
			Number: cNum,
			Title:  text,
			URL:    mangapillBase + href,
			IsNew:  true,
		})
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangapill: visit %s: %w", pageURL, err)
	}
	c.Wait()

	return chapters, nil
}

func extractMangaPillChapterNum(text, href string) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err == nil {
		return n
	}

	text = strings.TrimPrefix(text, "Chapter ")
	text = strings.TrimPrefix(text, "#")
	n, err = strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err == nil {
		return n
	}

	return math.NaN()
}

func (a *MangaPillAdapter) FetchPages(ctx context.Context, chapterUrl string) ([]string, error) {
	// TODO: implement
	return nil, nil
}
