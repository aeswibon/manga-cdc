package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMangaDexAdapter_FetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{
					"id": "abc-123",
					"attributes": {
						"title": {"en": "Test Manga"},
						"altTitles": [{"ja": "テスト漫画"}],
						"description": {"en": "A test manga"},
						"status": "ongoing"
					}
				},
				{
					"id": "def-456",
					"attributes": {
						"title": {"ja": "別の漫画"},
						"altTitles": [],
						"description": {},
						"status": "completed"
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(series))
	}
	if series[0].SourceID != "abc-123" || series[0].Title != "Test Manga" {
		t.Errorf("unexpected first series: %+v", series[0])
	}
	if series[0].SourceURL != "https://mangadex.org/title/abc-123" {
		t.Errorf("expected source URL, got %q", series[0].SourceURL)
	}
	if series[0].Status != "ONGOING" {
		t.Errorf("expected ONGOING, got %s", series[0].Status)
	}
	if series[0].IsActive != true {
		t.Errorf("expected IsActive=true")
	}
	if series[1].Title != "別の漫画" {
		t.Errorf("expected title from non-en fallback, got %s", series[1].Title)
	}
	if series[1].Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", series[1].Status)
	}
}

func TestMangaDexAdapter_FetchChapters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{
					"id": "ch1",
					"attributes": {
						"chapter": "1",
						"title": "Chapter One",
						"publishAt": "2024-01-15T00:00:00Z"
					}
				},
				{
					"id": "ch2",
					"attributes": {
						"chapter": "2.5",
						"title": null,
						"publishAt": ""
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	chapters, err := adapter.FetchChapters(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Number != 1 || chapters[0].Title != "Chapter One" {
		t.Errorf("unexpected first chapter: %+v", chapters[0])
	}
	if chapters[0].ReleaseDate.IsZero() {
		t.Error("expected non-zero release date")
	}
	if chapters[1].Number != 2.5 || chapters[1].Title != "" {
		t.Errorf("unexpected second chapter: %+v", chapters[1])
	}
	if !chapters[1].ReleaseDate.IsZero() {
		t.Error("expected zero release date for empty string")
	}
	if chapters[0].IsNew != true || chapters[1].IsNew != true {
		t.Error("expected IsNew=true for all chapters")
	}
}

func TestMangaDexAdapter_FetchSeries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manga/abc-123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": {
				"id": "abc-123",
				"attributes": {
					"title": {"en": "Test Manga"},
					"altTitles": [{"ja": "テスト"}],
					"description": {"en": "A test manga"},
					"status": "ongoing"
				},
				"relationships": [
					{"id": "author-1", "type": "author", "attributes": {"name": "Test Author"}},
					{"id": "cover-1", "type": "cover_art", "attributes": {"fileName": "cover.jpg"}}
				]
			}
		}`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	series, err := adapter.FetchSeries(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if series.Title != "Test Manga" {
		t.Errorf("unexpected title: %q", series.Title)
	}
	if series.Author != "Test Author" {
		t.Errorf("unexpected author: %q", series.Author)
	}
	if series.CoverURL != "https://uploads.mangadex.org/covers/abc-123/cover.jpg" {
		t.Errorf("unexpected cover: %q", series.CoverURL)
	}
	if series.Status != "ONGOING" {
		t.Errorf("unexpected status: %q", series.Status)
	}
}

func TestMangaDexAdapter_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	_, err := adapter.FetchLatest(context.Background())
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestMangaDexAdapter_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	_, err := adapter.FetchLatest(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMangaDexAdapter_ChapterNumberParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1", 1},
		{"2.5", 2.5},
		{"0", 0},
		{"", 0},
		{"abc", 0},
	}
	for _, tc := range tests {
		got := parseChapterNumber(tc.input)
		if got != tc.expected && !(tc.input == "abc" && got != got) {
			t.Errorf("parseChapterNumber(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestMangaDexAdapter_MalformedChapterNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{
					"id": "ch-bad",
					"attributes": {
						"chapter": "not-a-number",
						"title": "Bad Chapter",
						"publishAt": "2024-01-15T00:00:00Z"
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	chapters, err := adapter.FetchChapters(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 0 {
		t.Errorf("expected 0 chapters for malformed number, got %d", len(chapters))
	}
}

func TestMangaDexAdapter_NewMangaDexAdapterWithClient(t *testing.T) {
	client := &http.Client{Timeout: 5}
	adapter := NewMangaDexAdapterWithClient(client, "")
	if adapter.client != client {
		t.Error("expected client to be the injected one")
	}
}

func TestMangaDexAdapter_AltTitlesHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{
					"id": "abc-123",
					"attributes": {
						"title": {"en": "Test Manga"},
						"altTitles": [
							{"ja": "テスト漫画"},
							{"ko": "테스트 만화"},
							{"zh": "测试漫画"}
						],
						"description": {"en": "A test manga"},
						"status": "ongoing"
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(series))
	}
	if len(series[0].AltTitles) != 3 {
		t.Errorf("expected 3 alt titles, got %d: %v", len(series[0].AltTitles), series[0].AltTitles)
	}
}

func TestMangaDexAdapter_NonEnglishFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{
					"id": "xyz",
					"attributes": {
						"title": {"ja": "漫画"},
						"altTitles": [],
						"description": {},
						"status": "ongoing"
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if series[0].Title != "漫画" {
		t.Errorf("expected title from non-en key '漫画', got %q", series[0].Title)
	}
}

func TestMangaDexAdapter_HiatusStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [
				{
					"id": "xyz",
					"attributes": {
						"title": {"en": "Hiatus Manga"},
						"altTitles": [],
						"description": {},
						"status": "hiatus"
					}
				},
				{
					"id": "xyz2",
					"attributes": {
						"title": {"en": "Cancelled Manga"},
						"altTitles": [],
						"description": {},
						"status": "cancelled"
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if series[0].Status != "HIATUS" {
		t.Errorf("expected HIATUS, got %s", series[0].Status)
	}
	if series[1].Status != "CANCELLED" {
		t.Errorf("expected CANCELLED, got %s", series[1].Status)
	}
}

func TestMangaDexAdapter_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	_, err := adapter.FetchLatest(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestMangaDexAdapter_EmptySeries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": []}`))
	}))
	defer srv.Close()

	adapter := NewMangaDexAdapterWithClient(srv.Client(), srv.URL)
	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 series, got %d", len(series))
	}
}
