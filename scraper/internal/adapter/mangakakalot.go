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

const mangakakalotBase = "https://www.mangakakalot.gg"

type MangaKakalotAdapter struct {
	client *colly.Collector
	log    *slog.Logger
}

func NewMangaKakalotAdapter() *MangaKakalotAdapter {
	c := colly.NewCollector(
		colly.AllowedDomains("www.mangakakalot.gg", "mangakakalot.gg"),
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*mangakakalot.gg*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})
	c.SetRequestTimeout(30 * time.Second)

	return &MangaKakalotAdapter{
		client: c,
		log:    slog.Default().With("adapter", "mangakakalot"),
	}
}

func (m *MangaKakalotAdapter) Name() string {
	return "mangakakalot"
}

func (m *MangaKakalotAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	pageURL := mangakakalotBase + "/manga-list?type=latest"
	var series []model.Series
	var mu sync.Mutex

	m.client.OnHTML(".manga-list .manga-item, .list-body .item", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a", "href")
		if link == "" {
			return
		}
		slug := strings.TrimPrefix(link, "/manga/")
		slug = strings.TrimSuffix(slug, "/")
		if slug == "" {
			return
		}

		title := e.ChildText(".item-title, .manga-name")
		title = strings.TrimSpace(title)
		if title == "" {
			return
		}

		img := e.ChildAttr("img", "src")
		coverURL := ""
		if img != "" {
			coverURL = img
		}

		series = append(series, model.Series{
			SourceID:  slug,
			SourceURL: mangakakalotBase + "/manga/" + slug,
			Title:     title,
			CoverURL:  coverURL,
			Status:    "ONGOING",
			IsActive:  true,
		})
	})

	if err := m.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangakakalot: visit %s: %w", pageURL, err)
	}
	m.client.Wait()

	return series, nil
}

func (m *MangaKakalotAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	pageURL := mangakakalotBase + "/manga/" + seriesID
	var chapters []model.Chapter
	var mu sync.Mutex

	m.client.OnHTML(".chapter-list .row, .list-chapter li, table.chapter-table tr", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		link := e.ChildAttr("a", "href")
		if link == "" {
			return
		}

		numText := e.ChildText(".chapter-number, .chapter-name, span:first-child")
		numText = strings.TrimSpace(numText)
		numText = strings.TrimPrefix(numText, "Chapter ")
		numText = strings.TrimPrefix(numText, "Ch.")
		numText = strings.TrimSpace(numText)

		chapterNum, err := strconv.ParseFloat(numText, 64)
		if err != nil {
			return
		}

		titleText := e.ChildText(".chapter-name, .chapter-title")
		titleText = strings.TrimSpace(titleText)
		titleText = strings.TrimPrefix(titleText, "Chapter "+numText+" ")
		titleText = strings.TrimPrefix(titleText, "Ch. "+numText+" ")
		titleText = strings.TrimSpace(titleText)

		chapterURL := link
		if !strings.HasPrefix(link, "http") {
			chapterURL = mangakakalotBase + link
		}

		chapters = append(chapters, model.Chapter{
			Number: chapterNum,
			Title:  titleText,
			URL:    chapterURL,
			IsNew:  true,
		})
	})

	if err := m.client.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("mangakakalot: visit %s: %w", pageURL, err)
	}
	m.client.Wait()

	return chapters, nil
}
