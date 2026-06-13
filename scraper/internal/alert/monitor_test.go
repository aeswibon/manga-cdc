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
	m := New(log, srv.URL, Config{ZeroResultThreshold: 3})

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
	m := New(log, "", Config{ZeroResultThreshold: 1})
	m.RecordScrape(context.Background(), "mangafire", 0)
}

func TestMonitor_RecordValidation_alertsHighRejectRate(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := New(log, srv.URL, Config{
		RejectRateThreshold: 0.5,
		RejectRateMinSample: 5,
	})

	ctx := context.Background()
	m.RecordValidation(ctx, "mangadex", 10, 6)
	if calls.Load() != 1 {
		t.Fatalf("expected reject-rate alert, got %d calls", calls.Load())
	}

	m.RecordValidation(ctx, "mangadex", 10, 6)
	if calls.Load() != 1 {
		t.Fatalf("expected no duplicate reject-rate alert, got %d calls", calls.Load())
	}

	m.RecordValidation(ctx, "mangadex", 10, 1)
	m.RecordValidation(ctx, "mangadex", 10, 6)
	if calls.Load() != 2 {
		t.Fatalf("expected alert after recovery, got %d calls", calls.Load())
	}
}

func TestMonitor_RecordValidation_skipsSmallSamples(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := New(log, srv.URL, Config{
		RejectRateThreshold: 0.5,
		RejectRateMinSample: 5,
	})

	m.RecordValidation(context.Background(), "mangadex", 4, 4)
	if calls.Load() != 0 {
		t.Fatalf("expected no alert below min sample, got %d calls", calls.Load())
	}
}
