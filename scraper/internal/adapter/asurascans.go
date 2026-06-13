package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter/httpclient"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/gocolly/colly/v2"
)

const asurascansBase = "https://asurascans.com"

type AsuraScansAdapter struct {
	log    *slog.Logger
	client *colly.Collector
}

func NewAsuraScansAdapter() *AsuraScansAdapter {
	return &AsuraScansAdapter{
		log: slog.Default().With("adapter", "asurascans"),
	}
}

func (a *AsuraScansAdapter) Name() string {
	return "asurascans"
}

func (a *AsuraScansAdapter) newCollector() *colly.Collector {
	if a.client != nil {
		return a.client
	}
	c := colly.NewCollector(
		colly.AllowedDomains("asurascans.com", "www.asurascans.com"),
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*asurascans.com*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})
	c.SetRequestTimeout(60 * time.Second) // Match FlareSolverr timeout
	
	// Inject FlareSolverr
	c.WithTransport(&httpclient.Transport{
		Client:          httpclient.New(),
		UseFlareSolverr: true, // AsuraScans requires FlareSolverr
	})

	configureHTMLCollector(c)
	return c
}

func (a *AsuraScansAdapter) SetCollector(c *colly.Collector) {
	a.client = c
}

func (a *AsuraScansAdapter) FetchSeries(ctx context.Context, seriesID string) (model.Series, error) {
	pageURL := asurascansBase + "/comics/" + seriesID
	c := a.newCollector()
	meta, err := scrapeOpenGraph(c, pageURL)
	if err != nil {
		return model.Series{}, fmt.Errorf("asurascans: %w", err)
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

func (a *AsuraScansAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	pageURL := asurascansBase + "/browse"
	seen := make(map[string]bool)
	var series []model.Series
	var mu sync.Mutex

	c := a.newCollector()

	c.OnHTML("a[href^=\"/comics/\"]", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		href := e.Attr("href")
		if href == "" || strings.Count(href, "/") > 3 {
			return
		}

		img := e.ChildAttr("img", "src")
		title := e.ChildAttr("img", "alt")
		title = strings.TrimSpace(title)
		if title == "" {
			title = e.Text
			title = strings.TrimSpace(title)
		}
		if title == "" {
			return
		}

		slug := strings.TrimPrefix(href, "/comics/")
		slug = strings.TrimSuffix(slug, "/")

		if seen[slug] {
			return
		}
		seen[slug] = true

		coverURL := ""
		if img != "" {
			coverURL = img
		}

		series = append(series, model.Series{
			SourceID:  slug,
			SourceURL: asurascansBase + href,
			Title:     title,
			CoverURL:  coverURL,
			Status:    "ONGOING",
			IsActive:  true,
		})
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("asurascans: visit %s: %w", pageURL, err)
	}
	c.Wait()

	return series, nil
}

func (a *AsuraScansAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	pageURL := asurascansBase + "/comics/" + seriesID
	seen := make(map[float64]bool)
	var chapters []model.Chapter
	var mu sync.Mutex

	c := a.newCollector()

	c.OnHTML("a[href*=\"/chapter/\"]", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		href := e.Attr("href")
		if href == "" {
			return
		}

		parts := strings.Split(href, "/chapter/")
		if len(parts) != 2 {
			return
		}
		numStr := strings.TrimRight(parts[1], "/")
		numStr = strings.TrimSpace(numStr)
		chapterNum, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return
		}

		if seen[chapterNum] {
			return
		}
		seen[chapterNum] = true

		chapterURL := asurascansBase + href

		chapters = append(chapters, model.Chapter{
			Number: chapterNum,
			URL:    chapterURL,
			IsNew:  true,
		})
	})

	if err := c.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("asurascans: visit %s: %w", pageURL, err)
	}
	c.Wait()

	return chapters, nil
}

func (a *AsuraScansAdapter) FetchPages(ctx context.Context, chapterUrl string) ([]string, error) {
	var pages []string
	var mu sync.Mutex

	c := a.newCollector()

	c.OnHTML("#readerarea img", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()
		
		src := e.Attr("src")
		if src == "" {
			src = e.Attr("data-src")
		}
		
		if src != "" && !strings.Contains(src, "discord") {
			pages = append(pages, strings.TrimSpace(src))
		}
	})

	if err := c.Visit(chapterUrl); err != nil {
		return nil, fmt.Errorf("asurascans: fetch pages %s: %w", chapterUrl, err)
	}
	c.Wait()

	if len(pages) == 0 {
		return nil, fmt.Errorf("asurascans: no pages found at %s", chapterUrl)
	}

	return pages, nil
}
