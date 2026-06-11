package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gocolly/colly/v2"
)

type rewriteTransport struct {
	mockURL string
	next    http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	u, _ := req2.URL.Parse(t.mockURL)
	u.Path = req2.URL.Path
	u.RawQuery = req2.URL.RawQuery
	req2.URL = u
	req2.Host = u.Host
	return t.next.RoundTrip(req2)
}

func newCollyForTest(srv *httptest.Server) *colly.Collector {
	c := colly.NewCollector()
	c.SetClient(&http.Client{
		Transport: &rewriteTransport{mockURL: srv.URL, next: http.DefaultTransport},
	})
	c.OnRequest(func(r *colly.Request) {
		// no-op: prevents nil map panic if OnRequest is called on a fresh collector
	})
	return c
}

func TestMangaFireAdapter_FetchLatest_Fixture(t *testing.T) {
	srv := fixtureServer(t, "mangafire_latest.html")
	defer srv.Close()

	adapter := NewMangaFireAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 series from fixture, got %d", len(series))
	}
}

func TestMangaFireAdapter_FetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<body>
<div class="original card-lg">
<div class="unit">
<a class="poster" href="/manga/test-manga/">
<img src="https://example.com/cover.jpg">
</a>
<div class="info">
<a>Test Manga</a>
</div>
</div>
<div class="unit">
<a class="poster" href="/manga/no-cover/">
<img src="">
</a>
<div class="info">
<a>No Cover Manga</a>
</div>
</div>
</div>
</body>
</html>`))
	}))
	defer srv.Close()

	adapter := NewMangaFireAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(series))
	}
	if series[0].SourceID != "test-manga" || series[0].Title != "Test Manga" {
		t.Errorf("unexpected first series: %+v", series[0])
	}
	if series[0].CoverURL != "https://example.com/cover.jpg" {
		t.Errorf("expected cover URL, got %q", series[0].CoverURL)
	}
	if series[1].SourceID != "no-cover" || series[1].Title != "No Cover Manga" {
		t.Errorf("unexpected second series: %+v", series[1])
	}
	if series[0].Status != "ONGOING" || !series[0].IsActive {
		t.Errorf("expected ONGOING and active: %+v", series[0])
	}
}

func TestMangaFireAdapter_FetchLatest_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body></body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaFireAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 series, got %d", len(series))
	}
}

func TestMangaFireAdapter_FetchLatest_NoPoster(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<body>
<div class="original card-lg">
<div class="unit">
<div class="info"><a>Orphan Text</a></div>
</div>
</div>
</div>
</body>
</html>`))
	}))
	defer srv.Close()

	adapter := NewMangaFireAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 series (no poster link), got %d", len(series))
	}
}

func TestMangaFireAdapter_FetchChapters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<body>
<ul class="list-body">
<li class="item" data-number="1">
<a href="/read/test-manga/en/chapter-1">Chapter 1</a>
</li>
<li class="item" data-number="2">
<a href="/read/test-manga/en/chapter-2">Chapter 2</a>
</li>
</ul>
</body>
</html>`))
	}))
	defer srv.Close()

	adapter := NewMangaFireAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "test-manga")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Number != 1 || chapters[0].URL != "https://mangafire.to/read/test-manga/en/chapter-1" {
		t.Errorf("unexpected first chapter: %+v", chapters[0])
	}
	if chapters[1].Number != 2 || chapters[1].URL != "https://mangafire.to/read/test-manga/en/chapter-2" {
		t.Errorf("unexpected second chapter: %+v", chapters[1])
	}
	if !chapters[0].IsNew || !chapters[1].IsNew {
		t.Error("expected IsNew=true")
	}
}

func TestMangaFireAdapter_FetchChapters_NonChapterLinksFiltered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<body>
<ul class="list-body">
<li class="item" data-number="1">
<a href="/read/test-manga/en/chapter-1">Chapter 1</a>
</li>
<li class="item" data-number="">
<a href="/read/test-manga/other">Other</a>
</li>
</ul>
</body>
</html>`))
	}))
	defer srv.Close()

	adapter := NewMangaFireAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "test-manga")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(chapters))
	}
}
