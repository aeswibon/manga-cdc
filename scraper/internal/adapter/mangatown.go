package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/gocolly/colly/v2"
)

const mangatownBase = "https://www.mangatown.com"

type MangaTownAdapter struct {
	log    *slog.Logger
	client *colly.Collector
}

func NewMangaTownAdapter() *MangaTownAdapter {
	return &MangaTownAdapter{
		log: slog.Default().With("adapter", "mangatown"),
	}
}

func (m *MangaTownAdapter) Name() string {
	return "mangatown"
}

func (m *MangaTownAdapter) newCollector() *colly.Collector {
	if m.client != nil {
		return m.client
	}
	c := colly.NewCollector(
		colly.AllowedDomains("www.mangatown.com", "mangatown.com"),
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*mangatown.com*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})
	c.SetRequestTimeout(30 * time.Second)
	configureHTMLCollector(c)
	return c
}

func (m *MangaTownAdapter) SetCollector(c *colly.Collector) {
	m.client = c
}

func (m *MangaTownAdapter) FetchSeries(ctx context.Context, seriesID string) (model.Series, error) {
	pageURL := mangatownBase + "/manga/" + seriesID
	c := m.newCollector()
	meta, err := scrapeOpenGraph(c, pageURL)
	if err != nil {
		return model.Series{}, fmt.Errorf("mangatown: %w", err)
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

func (m *MangaTownAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	pageURL := mangatownBase + "/latest/"
	seen := make(map[string]bool)
	var series []model.Series
	var mu sync.Mutex

	c := m.newCollector()

	c.OnHTML("img[alt]", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		alt := e.Attr("alt")
		src := e.Attr("src")
		if alt == "" || strings.HasPrefix(alt, "Ad") {
			return
		}

		parent := e.DOM.Parent()
		link, _ := parent.Attr("href")
		if link == "" {
			link, _ = parent.Parent().Attr("href")
		}
		if link == "" || !strings.HasPrefix(link, "/manga/") {
			return
		}

		parts := strings.Split(strings.Trim(link, "/"), "/")
		if len(parts) < 2 {
			return
		}
		slug := parts[1]

		if seen[slug] {
			return
		}
		seen[slug] = true

		series = append(series, model.Series{
			SourceID:  slug,
			SourceURL: mangatownBase + "/manga/" + slug,
			Title:     alt,
			CoverURL:  src,
			Status:    "ONGOING",
			IsActive:  true,
		})
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangatown: visit %s: %w", pageURL, err)
	}
	c.Wait()

	return series, nil
}

func (m *MangaTownAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	pageURL := mangatownBase + "/manga/" + seriesID
	var chapters []model.Chapter
	var mu sync.Mutex

	c := m.newCollector()

	c.OnHTML("ul.chapter_list li", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a", "href")
		text := strings.TrimSpace(e.ChildText("a"))
		if link == "" || text == "" {
			return
		}

		if !strings.Contains(link, "/c") {
			return
		}

		cNum := extractMangaTownChapterNum(link)
		if math.IsNaN(cNum) {
			return
		}

		chapters = append(chapters, model.Chapter{
			Number: cNum,
			Title:  text,
			URL:    mangatownBase + link,
			IsNew:  true,
		})
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangatown: visit %s: %w", pageURL, err)
	}
	c.Wait()

	return chapters, nil
}

func extractMangaTownChapterNum(link string) float64 {
	parts := strings.Split(strings.TrimRight(link, "/"), "/")
	last := parts[len(parts)-1]
	if strings.HasPrefix(last, "v") {
		volParts := strings.Split(last, "/")
		last = volParts[len(volParts)-1]
	}
	last = strings.TrimPrefix(last, "c")
	n, err := strconv.ParseFloat(last, 64)
	if err != nil {
		return math.NaN()
	}
	return n
}
