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

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter/httpclient"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

const mangadexAPI = "https://api.mangadex.org"

type MangaDexAdapter struct {
	client  *http.Client
	baseURL string
}

func NewMangaDexAdapter() *MangaDexAdapter {
	return &MangaDexAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &httpclient.Transport{
				Client:          httpclient.New(),
				UseFlareSolverr: false,
			},
		},
		baseURL: mangadexAPI,
	}
}

func NewMangaDexAdapterWithClient(client *http.Client, baseURL string) *MangaDexAdapter {
	if baseURL == "" {
		baseURL = mangadexAPI
	}
	return &MangaDexAdapter{
		client:  client,
		baseURL: baseURL,
	}
}

func (m *MangaDexAdapter) Name() string {
	return "mangadex"
}

func setMangaDexHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "manga-cdc-scraper/1.0")
	req.Header.Set("Referer", "https://mangadex.org/")
}

func mangadexLocalizedText(values map[string]string) string {
	for _, key := range []string{"en", "ja-ro", "ja"} {
		if v := strings.TrimSpace(values[key]); v != "" {
			return v
		}
	}
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type mangadexMangaList struct {
	Data []struct {
		ID            string `json:"id"`
		Relationships []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"relationships"`
		Attributes struct {
			Title       map[string]string   `json:"title"`
			AltTitles   []map[string]string `json:"altTitles"`
			Description map[string]string   `json:"description"`
			Status      string              `json:"status"`
		} `json:"attributes"`
	} `json:"data"`
	Included []mangadexIncluded `json:"included"`
}

type mangadexIncluded struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		FileName string `json:"fileName"`
	} `json:"attributes"`
}

func mangadexCoverURL(mangaID, fileName string) string {
	if mangaID == "" || fileName == "" {
		return ""
	}
	return fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", mangaID, fileName)
}

func buildMangaDexCoverMap(included []mangadexIncluded) map[string]string {
	covers := make(map[string]string)
	for _, item := range included {
		if item.Type != "cover_art" {
			continue
		}
		covers[item.ID] = item.Attributes.FileName
	}
	return covers
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
	url := fmt.Sprintf("%s/manga?limit=20&order[updatedAt]=desc&availableTranslatedLanguage[]=en&includes[]=cover_art", m.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mangadex: create request: %w", err)
	}
	setMangaDexHeaders(req)

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
	coverFiles := buildMangaDexCoverMap(list.Included)
	for _, d := range list.Data {
		title := mangadexLocalizedText(d.Attributes.Title)

		altTitles := make([]string, 0, len(d.Attributes.AltTitles))
		for _, at := range d.Attributes.AltTitles {
			for _, v := range at {
				altTitles = append(altTitles, v)
			}
		}

		entry := d
		desc := mangadexLocalizedText(entry.Attributes.Description)

		coverURL := ""
		for _, rel := range entry.Relationships {
			if rel.Type != "cover_art" {
				continue
			}
			if fileName, ok := coverFiles[rel.ID]; ok {
				coverURL = mangadexCoverURL(entry.ID, fileName)
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
			SourceID:    entry.ID,
			Title:       title,
			AltTitles:   altTitles,
			Description: desc,
			CoverURL:    coverURL,
			SourceURL:   fmt.Sprintf("https://mangadex.org/title/%s", entry.ID),
			Status:      status,
			IsActive:    true,
		})
	}

	return series, nil
}

func (m *MangaDexAdapter) FetchSeries(ctx context.Context, seriesID string) (model.Series, error) {
	url := fmt.Sprintf("%s/manga/%s?includes[]=cover_art&includes[]=author&includes[]=artist", m.baseURL, seriesID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return model.Series{}, fmt.Errorf("mangadex: create series request: %w", err)
	}
	setMangaDexHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return model.Series{}, fmt.Errorf("mangadex: fetch series: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.Series{}, fmt.Errorf("mangadex: unexpected series status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.Series{}, fmt.Errorf("mangadex: read series body: %w", err)
	}

	var detail struct {
		Data struct {
			ID            string `json:"id"`
			Relationships []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Attributes struct {
					Name     string `json:"name"`
					FileName string `json:"fileName"`
				} `json:"attributes"`
			} `json:"relationships"`
			Attributes struct {
				Title       map[string]string   `json:"title"`
				AltTitles   []map[string]string `json:"altTitles"`
				Description map[string]string   `json:"description"`
				Status      string              `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
		Included []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name     string `json:"name"`
				FileName string `json:"fileName"`
			} `json:"attributes"`
		} `json:"included"`
	}
	if err := json.Unmarshal(body, &detail); err != nil {
		return model.Series{}, fmt.Errorf("mangadex: parse series: %w", err)
	}

	entry := detail.Data
	title := mangadexLocalizedText(entry.Attributes.Title)
	desc := mangadexLocalizedText(entry.Attributes.Description)

	namesByID := make(map[string]string)
	coverFiles := make(map[string]string)
	for _, item := range detail.Included {
		switch item.Type {
		case "author", "artist":
			namesByID[item.ID] = item.Attributes.Name
		case "cover_art":
			coverFiles[item.ID] = item.Attributes.FileName
		}
	}

	var authors, artists []string
	coverURL := ""
	seenAuthor := make(map[string]struct{})
	seenArtist := make(map[string]struct{})
	for _, rel := range entry.Relationships {
		switch rel.Type {
		case "author":
			name := strings.TrimSpace(rel.Attributes.Name)
			if name == "" {
				name = namesByID[rel.ID]
			}
			if name == "" {
				name = m.fetchCreatorName(ctx, "author", rel.ID)
			}
			if name != "" {
				if _, ok := seenAuthor[name]; !ok {
					authors = append(authors, name)
					seenAuthor[name] = struct{}{}
				}
			}
		case "artist":
			name := strings.TrimSpace(rel.Attributes.Name)
			if name == "" {
				name = namesByID[rel.ID]
			}
			if name == "" {
				name = m.fetchCreatorName(ctx, "artist", rel.ID)
			}
			if name != "" {
				if _, ok := seenArtist[name]; !ok {
					artists = append(artists, name)
					seenArtist[name] = struct{}{}
				}
			}
		case "cover_art":
			if coverURL != "" {
				continue
			}
			fileName := strings.TrimSpace(rel.Attributes.FileName)
			if fileName == "" {
				fileName = coverFiles[rel.ID]
			}
			if fileName == "" {
				fileName = m.fetchCoverFileName(ctx, rel.ID)
			}
			if fileName != "" {
				coverURL = mangadexCoverURL(entry.ID, fileName)
			}
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

	altTitles := make([]string, 0, len(entry.Attributes.AltTitles))
	for _, at := range entry.Attributes.AltTitles {
		for _, v := range at {
			altTitles = append(altTitles, v)
		}
	}

	return model.Series{
		SourceID:    entry.ID,
		Title:       title,
		AltTitles:   altTitles,
		Author:      strings.Join(authors, ", "),
		Artist:      strings.Join(artists, ", "),
		Description: desc,
		CoverURL:    coverURL,
		Status:      status,
		SourceURL:   fmt.Sprintf("https://mangadex.org/title/%s", entry.ID),
	}, nil
}

func (m *MangaDexAdapter) fetchCreatorName(ctx context.Context, entityType, id string) string {
	url := fmt.Sprintf("%s/%s/%s", m.baseURL, entityType, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	setMangaDexHeaders(req)
	resp, err := m.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var payload struct {
		Data struct {
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.Data.Attributes.Name
}

func (m *MangaDexAdapter) fetchCoverFileName(ctx context.Context, coverID string) string {
	url := fmt.Sprintf("%s/cover/%s", m.baseURL, coverID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	setMangaDexHeaders(req)
	resp, err := m.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var payload struct {
		Data struct {
			Attributes struct {
				FileName string `json:"fileName"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.Data.Attributes.FileName
}

func (m *MangaDexAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	url := fmt.Sprintf("%s/manga/%s/feed?limit=50&translatedLanguage[]=en&order[chapter]=desc", m.baseURL, seriesID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mangadex: create chapter request: %w", err)
	}
	setMangaDexHeaders(req)

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

func (a *MangaDexAdapter) FetchPages(ctx context.Context, chapterUrl string) ([]string, error) {
	// Extract chapter ID from URL
	// Example: https://mangadex.org/chapter/12345-uuid
	parts := strings.Split(chapterUrl, "/chapter/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mangadex chapter url: %s", chapterUrl)
	}
	chapterID := parts[1]

	url := fmt.Sprintf("%s/at-home/server/%s", a.baseURL, chapterID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mangadex: create athome request: %w", err)
	}
	setMangaDexHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex: fetch athome: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex: unexpected athome status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mangadex: read athome body: %w", err)
	}

	var atHome struct {
		BaseUrl string `json:"baseUrl"`
		Chapter struct {
			Hash string   `json:"hash"`
			Data []string `json:"data"`
		} `json:"chapter"`
	}

	if err := json.Unmarshal(body, &atHome); err != nil {
		return nil, fmt.Errorf("mangadex: parse athome: %w", err)
	}

	var pages []string
	for _, filename := range atHome.Chapter.Data {
		pages = append(pages, fmt.Sprintf("%s/data/%s/%s", atHome.BaseUrl, atHome.Chapter.Hash, filename))
	}

	return pages, nil
}
