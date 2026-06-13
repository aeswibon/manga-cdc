package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gocolly/colly/v2"
)

func TestScrapeOpenGraph(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><head>
<meta property="og:title" content="Test Series">
<meta property="og:description" content="A description">
<meta property="og:image" content="https://example.com/cover.jpg">
</head><body><h1>Ignored when OG present</h1></body></html>`))
	}))
	defer srv.Close()

	c := colly.NewCollector()
	c.SetClient(srv.Client())
	configureHTMLCollector(c)

	meta, err := scrapeOpenGraph(c, srv.URL)
	if err != nil {
		t.Fatalf("scrapeOpenGraph: %v", err)
	}
	if meta.Title != "Test Series" {
		t.Errorf("title = %q", meta.Title)
	}
	if meta.Description != "A description" {
		t.Errorf("description = %q", meta.Description)
	}
	if meta.Image != "https://example.com/cover.jpg" {
		t.Errorf("image = %q", meta.Image)
	}
}

func TestMangaFireAdapter_FetchSeries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><head>
<meta property="og:title" content="Fire Manga">
<meta property="og:image" content="https://example.com/fire.jpg">
</head></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaFireAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchSeries(context.Background(), "fire-manga")
	if err != nil {
		t.Fatalf("FetchSeries: %v", err)
	}
	if series.Title != "Fire Manga" {
		t.Errorf("title = %q", series.Title)
	}
	if series.CoverURL != "https://example.com/fire.jpg" {
		t.Errorf("cover = %q", series.CoverURL)
	}
	if series.SourceURL != mangafireBase+"/manga/fire-manga" {
		t.Errorf("source_url = %q", series.SourceURL)
	}
}
