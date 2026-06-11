package alert

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestMonitor_RecordScrape_alertsAfterThreshold(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Fatal("expected webhook body")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := New(log, srv.URL, 3)

	ctx := context.Background()
	m.RecordScrape(ctx, "mangadex", 0)
	m.RecordScrape(ctx, "mangadex", 0)
	if calls.Load() != 0 {
		t.Fatalf("expected no alert before threshold, got %d", calls.Load())
	}

	m.RecordScrape(ctx, "mangadex", 0)
	if calls.Load() != 1 {
		t.Fatalf("expected 1 alert at threshold, got %d", calls.Load())
	}

	m.RecordScrape(ctx, "mangadex", 0)
	if calls.Load() != 1 {
		t.Fatalf("expected no duplicate alert, got %d", calls.Load())
	}

	m.RecordScrape(ctx, "mangadex", 2)
	m.RecordScrape(ctx, "mangadex", 0)
	m.RecordScrape(ctx, "mangadex", 0)
	m.RecordScrape(ctx, "mangadex", 0)
	if calls.Load() != 2 {
		t.Fatalf("expected alert after recovery, got %d", calls.Load())
	}
}

func TestMonitor_RecordScrape_noWebhook(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := New(log, "", 1)
	m.RecordScrape(context.Background(), "mangafire", 0)
}
