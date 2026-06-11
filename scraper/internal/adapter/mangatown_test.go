package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMangaTownAdapter_FetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<div class="pic_box">
<a href="/manga/test-series/">
<img src="https://example.com/cover.jpg" alt="Test Series">
</a>
</div>
<div class="pic_box">
<a href="/manga/another-series/">
<img src="https://example.com/cover2.jpg" alt="Another Series">
</a>
</div>
<div class="pic_box">
<a href="/manga/another-series/">
<img src="https://example.com/cover2.jpg" alt="Duplicate">
</a>
</div>
</body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaTownAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 series (deduped), got %d", len(series))
	}
	if series[0].SourceID != "test-series" || series[0].Title != "Test Series" {
		t.Errorf("unexpected first series: %+v", series[0])
	}
	if series[1].SourceID != "another-series" || series[1].Title != "Another Series" {
		t.Errorf("unexpected second series: %+v", series[1])
	}
	if series[0].Status != "ONGOING" || !series[0].IsActive {
		t.Errorf("expected ONGOING and active: %+v", series[0])
	}
}

func TestMangaTownAdapter_FetchLatest_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body></body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaTownAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 series, got %d", len(series))
	}
}

func TestMangaTownAdapter_FetchLatest_AdFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<div class="pic_box">
<a href="/manga/real-series/">
<img src="https://example.com/c1.jpg" alt="Real Series">
</a>
</div>
<div class="pic_box">
<img src="https://example.com/ad.jpg" alt="Ad Banner">
</div>
</body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaTownAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 series (ad filtered), got %d", len(series))
	}
	if series[0].SourceID != "real-series" {
		t.Errorf("expected real-series, got %s", series[0].SourceID)
	}
}

func TestMangaTownAdapter_FetchChapters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<ul class="chapter_list">
<li><a href="/manga/test-series/c001/">Chapter 1</a></li>
<li><a href="/manga/test-series/c002/">Chapter 2</a></li>
</ul>
</body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaTownAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "test-series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Number != 1 || chapters[0].Title != "Chapter 1" {
		t.Errorf("unexpected first chapter: %+v", chapters[0])
	}
	if chapters[1].Number != 2 || chapters[1].Title != "Chapter 2" {
		t.Errorf("unexpected second chapter: %+v", chapters[1])
	}
	if !chapters[0].IsNew || !chapters[1].IsNew {
		t.Error("expected IsNew=true")
	}
}

func TestMangaTownAdapter_FetchChapters_VolumeFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<ul class="chapter_list">
<li><a href="/manga/test-series/v01/c001/">Vol 1 Ch 1</a></li>
<li><a href="/manga/test-series/v01/c002/">Vol 1 Ch 2</a></li>
</ul>
</body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaTownAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "test-series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Number != 1 {
		t.Errorf("expected chapter 1, got %f", chapters[0].Number)
	}
	if chapters[1].Number != 2 {
		t.Errorf("expected chapter 2, got %f", chapters[1].Number)
	}
}

func TestMangaTownAdapter_FetchChapters_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body></body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaTownAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "test-series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(chapters))
	}
}

func TestExtractMangaTownChapterNum(t *testing.T) {
	tests := []struct {
		link     string
		expected float64
	}{
		{"/manga/series/c001/", 1},
		{"/manga/series/c358/", 358},
		{"/manga/series/v01/c001/", 1},
		{"/manga/series/v12/c245/", 245},
		{"/manga/series/c029/", 29},
	}
	for _, tc := range tests {
		got := extractMangaTownChapterNum(tc.link)
		if got != tc.expected {
			t.Errorf("extractMangaTownChapterNum(%q) = %v, want %v", tc.link, got, tc.expected)
		}
	}
}
