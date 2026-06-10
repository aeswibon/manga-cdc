package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

const mangaplusAPI = "https://jumpg-web-api.tokyo-cdn.com/api"

type MangaPlusAdapter struct {
	client *http.Client
}

func NewMangaPlusAdapter() *MangaPlusAdapter {
	return &MangaPlusAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *MangaPlusAdapter) Name() string {
	return "mangaplus"
}

type mangaPlusResponse struct {
	Success struct {
		AllView struct {
			Titles []struct {
				TitleID     int    `json:"titleId"`
				Name        string `json:"name"`
				Author      string `json:"author"`
				PortraitURL string `json:"portraitUrl"`
			} `json:"titles"`
		} `json:"allView"`
	} `json:"success"`
}

func (m *MangaPlusAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	url := fmt.Sprintf("%s/title_list/all", mangaplusAPI)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: create request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangaplus: unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: read body: %w", err)
	}

	var result mangaPlusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("mangaplus: parse: %w", err)
	}

	if result.Success.AllView.Titles == nil {
		return nil, fmt.Errorf("mangaplus: empty response")
	}

	var series []model.Series
	for _, t := range result.Success.AllView.Titles {
		series = append(series, model.Series{
			SourceID: fmt.Sprintf("%d", t.TitleID),
			Title:    t.Name,
			Author:   t.Author,
			CoverURL: t.PortraitURL,
			Status:   "ONGOING",
			IsActive: true,
		})
	}

	return series, nil
}

type mangaPlusChapterResponse struct {
	Success struct {
		TitleDetailView struct {
			TitleID  int    `json:"titleId"`
			Name     string `json:"name"`
			Author   string `json:"author"`
			Chapters []struct {
				ChapterID     int     `json:"chapterId"`
				Name          string  `json:"name"`
				ChapterNumber float64 `json:"chapterNumber,string"`
				StartTime     string  `json:"startTime"`
			} `json:"chapters"`
		} `json:"titleDetailView"`
	} `json:"success"`
}

func (m *MangaPlusAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	url := fmt.Sprintf("%s/title_detail?title_id=%s", mangaplusAPI, seriesID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: create chapter request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: fetch chapters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangaplus: unexpected chapter status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: read chapter body: %w", err)
	}

	var result mangaPlusChapterResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("mangaplus: parse chapters: %w", err)
	}

	if result.Success.TitleDetailView.Chapters == nil {
		return nil, nil
	}

	var chapters []model.Chapter
	for _, c := range result.Success.TitleDetailView.Chapters {
		releaseDate, _ := time.Parse(time.RFC3339, c.StartTime)

		chapters = append(chapters, model.Chapter{
			Number:      c.ChapterNumber,
			Title:       c.Name,
			URL:         fmt.Sprintf("https://mangaplus.shueisha.co.jp/viewer/%d", c.ChapterID),
			ReleaseDate: releaseDate,
			IsNew:       true,
		})
	}

	return chapters, nil
}
