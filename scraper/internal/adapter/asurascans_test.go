package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAsuraScansAdapter_FetchLatest_Fixture(t *testing.T) {
	srv := fixtureServer(t, "asurascans_latest.html")
	defer srv.Close()

	adapter := NewAsuraScansAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 series from fixture, got %d", len(series))
	}
}

func TestAsuraScansAdapter_FetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<body>
<div class="listupd">
<a href="/comics/test-series/">
<img src="https://example.com/cover.jpg" alt="Test Series">
</a>
<a href="/comics/another-series/">
<img src="https://example.com/cover2.jpg" alt="Another Series">
</a>
<a href="/comics/test-series/">
<img src="https://example.com/cover.jpg" alt="Duplicate">
</a>
</div>
</body>
</html>`))
	}))
	defer srv.Close()

	adapter := NewAsuraScansAdapter()
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

func TestAsuraScansAdapter_FetchLatest_NoCover(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<body>
<a href="/comics/no-cover/">
<img src="" alt="No Cover">
</a>
</body>
</html>`))
	}))
	defer srv.Close()

	adapter := NewAsuraScansAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(series))
	}
	if series[0].CoverURL != "" {
		t.Errorf("expected empty cover URL, got %q", series[0].CoverURL)
	}
}

func TestAsuraScansAdapter_FetchLatest_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body></body></html>`))
	}))
	defer srv.Close()

	adapter := NewAsuraScansAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 series, got %d", len(series))
	}
}

func TestAsuraScansAdapter_FetchChapters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><div class="chlist"><a href="/comics/test-series/chapter/1">Chapter 1</a><a href="/comics/test-series/chapter/2">Chapter 2</a><a href="/comics/test-series/chapter/1">Duplicate</a></div></body></html>`))
	}))
	defer srv.Close()

	adapter := NewAsuraScansAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "test-series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters (deduped), got %d", len(chapters))
	}
	if chapters[0].Number != 1 || chapters[0].URL != "https://asurascans.com/comics/test-series/chapter/1" {
		t.Errorf("unexpected first chapter: %+v", chapters[0])
	}
	if chapters[1].Number != 2 || chapters[1].URL != "https://asurascans.com/comics/test-series/chapter/2" {
		t.Errorf("unexpected second chapter: %+v", chapters[1])
	}
	if !chapters[0].IsNew || !chapters[1].IsNew {
		t.Error("expected IsNew=true")
	}
}

func TestAsuraScansAdapter_FetchChapters_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body></body></html>`))
	}))
	defer srv.Close()

	adapter := NewAsuraScansAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "test-series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(chapters))
	}
}

func TestAsuraScansAdapter_FetchLatest_EmptyTitleFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<body>
<a href="/comics/fallback-series/">
<img src="" alt="">
Fallback Title
</a>
</body>
</html>`))
	}))
	defer srv.Close()

	adapter := NewAsuraScansAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(series))
	}
	if series[0].Title != "Fallback Title" {
		t.Errorf("expected 'Fallback Title', got %q", series[0].Title)
	}
}

func TestAsuraScansAdapter_FetchLatest_TooDeepHref(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<body>
<a href="/comics/series/chapter/1">Deep Link</a>
<a href="/comics/series/">Normal Link</a>
</body>
</html>`))
	}))
	defer srv.Close()

	adapter := NewAsuraScansAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 series (deep href filtered), got %d", len(series))
	}
}
