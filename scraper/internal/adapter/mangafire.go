package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/gocolly/colly/v2"
)

const mangafireBase = "https://mangafire.to"

type MangaFireAdapter struct {
	client *colly.Collector
	log    *slog.Logger
}

func NewMangaFireAdapter() *MangaFireAdapter {
	c := colly.NewCollector(
		colly.AllowedDomains("mangafire.to"),
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*mangafire.to*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})
	c.SetRequestTimeout(30 * time.Second)

	return &MangaFireAdapter{
		client: c,
		log:    slog.Default().With("adapter", "mangafire"),
	}
}

func (m *MangaFireAdapter) Name() string {
	return "mangafire"
}

func (m *MangaFireAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	pageURL := mangafireBase + "/home"
	var series []model.Series
	var mu sync.Mutex

	m.client.OnHTML(".original.card-lg .unit", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a.poster", "href")
		if link == "" {
			return
		}
		slug := strings.TrimPrefix(link, "/manga/")
		slug = strings.TrimSuffix(slug, "/")
		if slug == "" {
			return
		}

		title := e.ChildText(".info a")
		title = strings.TrimSpace(title)
		if title == "" {
			return
		}

		img := e.ChildAttr("a.poster img", "src")
		coverURL := ""
		if img != "" {
			coverURL = img
		}

		series = append(series, model.Series{
			SourceID:  slug,
			SourceURL: mangafireBase + "/manga/" + slug,
			Title:     title,
			CoverURL:  coverURL,
			Status:    "ONGOING",
			IsActive:  true,
		})
	})

	if err := m.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangafire: visit %s: %w", pageURL, err)
	}
	m.client.Wait()

	return series, nil
}

func (m *MangaFireAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	pageURL := mangafireBase + "/manga/" + seriesID
	var chapters []model.Chapter
	var mu sync.Mutex

	m.client.OnHTML(".original.card-lg .unit .content[data-name=chap] li", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a", "href")
		if link == "" || !strings.Contains(link, "/en/chapter-") {
			return
		}

		text := e.ChildText("span:first-child")
		text = strings.TrimSpace(text)

		numStr := strings.TrimPrefix(text, "Chap ")
		numStr = strings.TrimSpace(numStr)
		chapterNum, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return
		}

		titleText := e.ChildText("span b")
		var chapterTitle string
		if titleText != "" && titleText != "EN" {
			chapterTitle = titleText
		}

		chapterURL := mangafireBase + link

		chapters = append(chapters, model.Chapter{
			Number: chapterNum,
			Title:  chapterTitle,
			URL:    chapterURL,
			IsNew:  true,
		})
	})

	if err := m.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangafire: visit %s: %w", pageURL, err)
	}
	m.client.Wait()

	return chapters, nil
}
