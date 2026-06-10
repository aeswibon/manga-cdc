package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

const mangadexAPI = "https://api.mangadex.org"

type MangaDexAdapter struct {
	client *http.Client
}

func NewMangaDexAdapter() *MangaDexAdapter {
	return &MangaDexAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *MangaDexAdapter) Name() string {
	return "mangadex"
}

type mangadexMangaList struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title       map[string]string   `json:"title"`
			AltTitles   []map[string]string `json:"altTitles"`
			Description map[string]string   `json:"description"`
			Status      string              `json:"status"`
		} `json:"attributes"`
	} `json:"data"`
}

func parseChapterNumber(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return n
}

func (m *MangaDexAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	url := fmt.Sprintf("%s/manga?limit=20&order[updatedAt]=desc&availableTranslatedLanguage[]=en", mangadexAPI)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mangadex: create request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex: unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mangadex: read body: %w", err)
	}

	var list mangadexMangaList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("mangadex: parse: %w", err)
	}

	var series []model.Series
	for _, d := range list.Data {
		title := d.Attributes.Title["en"]
		if title == "" {
			for _, t := range d.Attributes.Title {
				title = t
				break
			}
		}

		altTitles := make([]string, 0, len(d.Attributes.AltTitles))
		for _, at := range d.Attributes.AltTitles {
			for _, v := range at {
				altTitles = append(altTitles, v)
			}
		}

		entry := d
		desc := entry.Attributes.Description["en"]
		if desc == "" {
			for _, val := range entry.Attributes.Description {
				desc = val
				break
			}
		}

		status := entry.Attributes.Status
		switch status {
		case "ongoing":
			status = "ONGOING"
		case "completed":
			status = "COMPLETED"
		case "hiatus":
			status = "HIATUS"
		case "cancelled":
			status = "CANCELLED"
		}

		series = append(series, model.Series{
			SourceID:  entry.ID,
			Title:     title,
			AltTitles: altTitles,
			Status:    status,
			IsActive:  true,
		})
	}

	return series, nil
}

func (m *MangaDexAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	url := fmt.Sprintf("%s/manga/%s/feed?limit=50&translatedLanguage[]=en&order[chapter]=desc", mangadexAPI, seriesID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mangadex: create chapter request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex: fetch chapters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex: unexpected chapter status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mangadex: read chapter body: %w", err)
	}

	var feed struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Chapter   string  `json:"chapter"`
				Title     *string `json:"title"`
				PublishAt string  `json:"publishAt"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("mangadex: parse chapters: %w", err)
	}

	var chapters []model.Chapter
	for _, d := range feed.Data {
		chapterNum := parseChapterNumber(d.Attributes.Chapter)
		if math.IsNaN(chapterNum) {
			continue
		}

		var title string
		if d.Attributes.Title != nil {
			title = *d.Attributes.Title
		}

		var releaseDate time.Time
		if d.Attributes.PublishAt != "" {
			releaseDate, _ = time.Parse(time.RFC3339, d.Attributes.PublishAt)
		}

		chapters = append(chapters, model.Chapter{
			Number:      chapterNum,
			Title:       title,
			URL:         fmt.Sprintf("https://mangadex.org/chapter/%s", d.ID),
			ReleaseDate: releaseDate,
			IsNew:       true,
		})
	}

	return chapters, nil
}
