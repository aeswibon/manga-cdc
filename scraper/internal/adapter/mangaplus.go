package adapter

import (
	"compress/gzip"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/adapter/mangapluspb"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"google.golang.org/protobuf/proto"
)

const (
	mangaplusAPI = "https://jumpg-api.tokyo-cdn.com/api"
	securityKey  = "4Kin9vGg"
	appVersion   = 237
	deviceID     = "manga-cdc-scraper-v1"
)

type MangaPlusAdapter struct {
	client  *http.Client
	secret  string
	mu      sync.Mutex
	log     *slog.Logger
	baseURL string
}

func NewMangaPlusAdapter() *MangaPlusAdapter {
	return &MangaPlusAdapter{
		client:  &http.Client{Timeout: 30 * time.Second},
		log:     slog.Default().With("adapter", "mangaplus"),
		baseURL: mangaplusAPI,
	}
}

func NewMangaPlusAdapterWithClient(client *http.Client, baseURL string) *MangaPlusAdapter {
	if baseURL == "" {
		baseURL = mangaplusAPI
	}
	return &MangaPlusAdapter{
		client:  client,
		log:     slog.Default().With("adapter", "mangaplus"),
		baseURL: baseURL,
	}
}

func (m *MangaPlusAdapter) SetSecret(secret string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secret = secret
}

func (m *MangaPlusAdapter) Name() string {
	return "mangaplus"
}

func (m *MangaPlusAdapter) register(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.secret != "" {
		return nil
	}

	deviceToken := fmt.Sprintf("%x", md5.Sum([]byte(deviceID)))
	securityKeyHash := fmt.Sprintf("%x", md5.Sum([]byte(deviceToken+securityKey)))

	params := url.Values{
		"device_token": {deviceToken},
		"security_key": {securityKeyHash},
		"os":           {"android"},
		"os_ver":       {"35"},
		"app_ver":      {strconv.Itoa(appVersion)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, m.baseURL+"/register?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("mangaplus: create register request: %w", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "okhttp/4.12.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("mangaplus: register request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readDecompressed(resp.Body, resp.Header)
	if err != nil {
		return fmt.Errorf("mangaplus: read register body: %w", err)
	}

	var response mangapluspb.Response
	if err := proto.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("mangaplus: unmarshal register response: %w", err)
	}

	success := response.GetSuccess()
	if success == nil {
		return fmt.Errorf("mangaplus: register failed: %v", response.GetError())
	}

	regData := success.GetRegisterationData()
	if regData == nil {
		return fmt.Errorf("mangaplus: register: no registration data")
	}

	m.secret = regData.GetDeviceSecret()
	m.log.Debug("device registered", "secret_prefix", m.secret[:8])
	return nil
}

func readDecompressed(r io.ReadCloser, header http.Header) ([]byte, error) {
	if header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gr.Close()
		return io.ReadAll(gr)
	}
	return io.ReadAll(r)
}

func (m *MangaPlusAdapter) doRequest(ctx context.Context, endpoint string, extraParams url.Values) ([]byte, error) {
	if err := m.register(ctx); err != nil {
		return nil, err
	}

	params := url.Values{
		"os":      {"android"},
		"os_ver":  {"35"},
		"app_ver": {strconv.Itoa(appVersion)},
		"secret":  {m.secret},
	}
	for k, v := range extraParams {
		params[k] = v
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL+endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: create request: %w", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "okhttp/4.12.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readDecompressed(resp.Body, resp.Header)
	if err != nil {
		return nil, fmt.Errorf("mangaplus: read body: %w", err)
	}

	return body, nil
}

func (m *MangaPlusAdapter) FetchLatest(ctx context.Context) ([]model.Series, error) {
	body, err := m.doRequest(ctx, "/title_list/all_v3", url.Values{
		"type":  {"serializing"},
		"lang":  {"eng"},
		"clang": {"eng"},
	})
	if err != nil {
		return nil, err
	}

	var response mangapluspb.Response
	if err := proto.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("mangaplus: unmarshal titles: %w", err)
	}

	success := response.GetSuccess()
	if success == nil {
		return nil, fmt.Errorf("mangaplus: titles request failed: %v", response.GetError())
	}

	view := success.GetSearchView()
	if view == nil {
		return nil, fmt.Errorf("mangaplus: no searchView in response")
	}

	var series []model.Series
	seen := make(map[int32]bool)
	for _, group := range view.GetAllTitlesGroup() {
		for _, t := range group.GetTitles() {
			if t.GetLanguage() != mangapluspb.Language_ENGLISH {
				continue
			}
			if seen[t.GetTitleId()] {
				continue
			}
			seen[t.GetTitleId()] = true
			series = append(series, model.Series{
				SourceID:  fmt.Sprintf("%d", t.GetTitleId()),
				Title:     t.GetName(),
				Author:    t.GetAuthor(),
				CoverURL:  t.GetPortraitImageUrl(),
				SourceURL: fmt.Sprintf("https://mangaplus.shueisha.co.jp/titles/%d", t.GetTitleId()),
				Status:    "ONGOING",
				IsActive:  true,
			})
		}
	}

	return series, nil
}

func (m *MangaPlusAdapter) fetchTitleDetail(ctx context.Context, seriesID string) (*mangapluspb.TitleDetailView, error) {
	body, err := m.doRequest(ctx, "/title_detailV3", url.Values{
		"title_id": {seriesID},
		"lang":     {"eng"},
		"clang":    {"eng"},
	})
	if err != nil {
		return nil, err
	}

	var response mangapluspb.Response
	if err := proto.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("mangaplus: unmarshal title detail: %w", err)
	}

	success := response.GetSuccess()
	if success == nil {
		return nil, fmt.Errorf("mangaplus: title detail request failed: %v", response.GetError())
	}

	detail := success.GetTitleDetailView()
	if detail == nil {
		return nil, fmt.Errorf("mangaplus: no titleDetailView in response")
	}
	return detail, nil
}

func (m *MangaPlusAdapter) FetchSeries(ctx context.Context, seriesID string) (model.Series, error) {
	detail, err := m.fetchTitleDetail(ctx, seriesID)
	if err != nil {
		return model.Series{}, err
	}

	title := detail.GetTitle()
	coverURL := detail.GetTitleImageUrl()
	if coverURL == "" && title != nil {
		coverURL = title.GetPortraitImageUrl()
	}

	name := ""
	author := ""
	if title != nil {
		name = title.GetName()
		author = title.GetAuthor()
	}

	return model.Series{
		SourceID:    seriesID,
		Title:       name,
		Author:      author,
		Description: detail.GetOverview(),
		CoverURL:    coverURL,
		SourceURL:   fmt.Sprintf("https://mangaplus.shueisha.co.jp/titles/%s", seriesID),
		Status:      "ONGOING",
	}, nil
}

func (m *MangaPlusAdapter) FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error) {
	detail, err := m.fetchTitleDetail(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	seen := make(map[int32]bool)

	var chapters []model.Chapter
	for _, c := range detail.GetChapterListV2() {
		if seen[c.GetChapterId()] {
			continue
		}
		seen[c.GetChapterId()] = true

		releaseDate := time.Unix(c.GetStartTimeStamp(), 0)
		chapterNumber := float64(c.GetChapterId())

		chapters = append(chapters, model.Chapter{
			Number:      chapterNumber,
			Title:       c.GetSubTitle(),
			URL:         fmt.Sprintf("https://mangaplus.shueisha.co.jp/viewer/%d", c.GetChapterId()),
			ReleaseDate: releaseDate,
			IsNew:       true,
		})
	}

	return chapters, nil
}

func (m *MangaPlusAdapter) FetchPages(ctx context.Context, chapterUrl string) ([]string, error) {
	parts := strings.Split(chapterUrl, "/viewer/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mangaplus chapter url: %s", chapterUrl)
	}
	chapterID := parts[1]

	body, err := m.doRequest(ctx, "/manga_viewer", url.Values{
		"chapter_id":  {chapterID},
		"split":       {"yes"},
		"img_quality": {"super_high"},
	})
	if err != nil {
		return nil, fmt.Errorf("mangaplus: fetch viewer: %w", err)
	}

	var response mangapluspb.Response
	if err := proto.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("mangaplus: unmarshal viewer: %w", err)
	}

	success := response.GetSuccess()
	if success == nil {
		return nil, fmt.Errorf("mangaplus: viewer request failed: %v", response.GetError())
	}

	viewer := success.GetMangaViewer()
	if viewer == nil {
		return nil, fmt.Errorf("mangaplus: no mangaViewer in response")
	}

	var pages []string
	for _, p := range viewer.GetPages() {
		if mp := p.GetMangaPage(); mp != nil && mp.GetImageUrl() != "" {
			pages = append(pages, mp.GetImageUrl())
		}
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("mangaplus: no pages found at %s", chapterUrl)
	}

	return pages, nil
}
