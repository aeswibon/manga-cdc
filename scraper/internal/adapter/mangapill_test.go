package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMangaPillAdapter_FetchLatest_Fixture(t *testing.T) {
	srv := fixtureServer(t, "mangapill_latest.html")
	defer srv.Close()

	adapter := NewMangaPillAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 series from fixture, got %d", len(series))
	}
}

func TestMangaPillAdapter_FetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<div class="rounded">
<a href="/chapters/1175-10213000/fairy-tail-100-years-quest-chapter-213">
<img data-src="https://cdn.example.com/i/1175.jpeg" alt="Fairy Tail: 100 Years Quest Chapter 213">
</a>
<div>
<a href="/chapters/1175-10213000/fairy-tail-100-years-quest-chapter-213"><div>#213</div></a>
<a href="/manga/1175/fairy-tail-100-years-quest"><div>Fairy Tail: 100 Years Quest</div></a>
</div>
</div>
<div class="rounded">
<a href="/chapters/5343-10241000/mokushiroku-no-yonkishi-chapter-241">
<img data-src="https://cdn.example.com/i/5343.jpeg" alt="Mokushiroku no Yonkishi Chapter 241">
</a>
<div>
<a href="/chapters/5343-10241000/mokushiroku-no-yonkishi-chapter-241"><div>#241</div></a>
<a href="/manga/5343/mokushiroku-no-yonkishi"><div>Mokushiroku no Yonkishi</div></a>
</div>
</div>
</body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaPillAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(series))
	}
	if series[0].SourceID != "1175/fairy-tail-100-years-quest" {
		t.Errorf("unexpected sourceID: %s", series[0].SourceID)
	}
	if series[0].Title != "Fairy Tail: 100 Years Quest" {
		t.Errorf("unexpected title: %s", series[0].Title)
	}
	if series[0].SourceURL != "https://mangapill.com/manga/1175/fairy-tail-100-years-quest" {
		t.Errorf("unexpected sourceURL: %s", series[0].SourceURL)
	}
	if series[0].CoverURL != "https://cdn.example.com/i/1175.jpeg" {
		t.Errorf("unexpected coverURL: %s", series[0].CoverURL)
	}
}

func TestMangaPillAdapter_FetchLatest_Dedup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<div class="rounded">
<a href="/chapters/1175-10213000/fairy-tail-100-years-quest-chapter-213">
<img data-src="https://cdn.example.com/i/1175.jpeg" alt="Fairy Tail: 100 Years Quest Chapter 213">
</a>
<div>
<a href="/chapters/1175-10213000/..."><div>#213</div></a>
<a href="/manga/1175/fairy-tail-100-years-quest"><div>Fairy Tail: 100 Years Quest</div></a>
</div>
</div>
<div class="rounded">
<a href="/chapters/1175-10212000/...">
<img data-src="https://cdn.example.com/i/1175.jpeg" alt="Fairy Tail: 100 Years Quest Chapter 212">
</a>
<div>
<a href="/chapters/1175-10212000/..."><div>#212</div></a>
<a href="/manga/1175/fairy-tail-100-years-quest"><div>Fairy Tail: 100 Years Quest</div></a>
</div>
</div>
</body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaPillAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 series (deduped), got %d", len(series))
	}
}

func TestMangaPillAdapter_FetchLatest_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body></body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaPillAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	series, err := adapter.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 series, got %d", len(series))
	}
}

func TestMangaPillAdapter_FetchChapters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<div id="chapters" class="p-3">
<div data-filter-list class="my-3 grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6">
<a class="border" href="/chapters/1175-10213000/slug-chapter-213" title=" Chapter 213">Chapter 213</a>
<a class="border" href="/chapters/1175-10212000/slug-chapter-212" title=" Chapter 212">Chapter 212</a>
<a class="border" href="/chapters/1175-10211000/slug-chapter-211" title=" Chapter 211">Chapter 211</a>
</div>
</div>
</body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaPillAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "1175/fairy-tail-100-years-quest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(chapters))
	}
	if chapters[0].Number != 213 || chapters[0].Title != "Chapter 213" {
		t.Errorf("unexpected first chapter: %+v", chapters[0])
	}
	if chapters[1].Number != 212 || chapters[1].Title != "Chapter 212" {
		t.Errorf("unexpected second chapter: %+v", chapters[1])
	}
	if chapters[2].Number != 211 || chapters[2].Title != "Chapter 211" {
		t.Errorf("unexpected third chapter: %+v", chapters[2])
	}
}

func TestMangaPillAdapter_FetchChapters_NumericOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
<div id="chapters" class="p-3">
<div data-filter-list class="my-3 grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6">
<a class="border" href="/chapters/1-213/slug-chapter-213">213</a>
<a class="border" href="/chapters/1-212/slug-chapter-212">212</a>
</div>
</div>
</body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaPillAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "1/test-series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Number != 213 {
		t.Errorf("expected 213, got %f", chapters[0].Number)
	}
	if chapters[1].Number != 212 {
		t.Errorf("expected 212, got %f", chapters[1].Number)
	}
}

func TestMangaPillAdapter_FetchChapters_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body></body></html>`))
	}))
	defer srv.Close()

	adapter := NewMangaPillAdapter()
	adapter.SetCollector(newCollyForTest(srv))

	chapters, err := adapter.FetchChapters(context.Background(), "1/test-series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(chapters))
	}
}

func TestExtractMangaPillChapterNum(t *testing.T) {
	tests := []struct {
		text     string
		href     string
		expected float64
	}{
		{"Chapter 213", "/chapters/1-213/slug-chapter-213", 213},
		{"#213", "/chapters/1-213/slug-chapter-213", 213},
		{"213", "/chapters/1-213/slug-chapter-213", 213},
		{"Chapter 1", "/chapters/1-1/slug-chapter-1", 1},
	}
	for _, tc := range tests {
		got := extractMangaPillChapterNum(tc.text, tc.href)
		if got != tc.expected {
			t.Errorf("extractMangaPillChapterNum(%q, %q) = %v, want %v", tc.text, tc.href, got, tc.expected)
		}
	}
}
