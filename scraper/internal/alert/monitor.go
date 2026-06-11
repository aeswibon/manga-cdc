package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var zeroResultCycles = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "scraper_zero_result_cycles",
	Help: "Consecutive scrape cycles where a source returned zero series",
}, []string{"source"})

var seriesFetchedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "scraper_series_fetched_total",
	Help: "Total series returned by source adapters per scrape cycle",
}, []string{"source"})

type Monitor struct {
	log       *slog.Logger
	webhook   string
	threshold int
	client    *http.Client

	mu      sync.Mutex
	sources map[string]*sourceState
}

type sourceState struct {
	zeroCycles int
	alerted    bool
}

func New(log *slog.Logger, webhookURL string, threshold int) *Monitor {
	if threshold < 1 {
		threshold = 3
	}
	return &Monitor{
		log:       log,
		webhook:   webhookURL,
		threshold: threshold,
		client:    &http.Client{Timeout: 10 * time.Second},
		sources:   make(map[string]*sourceState),
	}
}

func (m *Monitor) RecordScrape(ctx context.Context, source string, seriesFetched int) {
	seriesFetchedTotal.WithLabelValues(source).Add(float64(seriesFetched))

	m.mu.Lock()
	state := m.sources[source]
	if state == nil {
		state = &sourceState{}
		m.sources[source] = state
	}

	if seriesFetched > 0 {
		state.zeroCycles = 0
		state.alerted = false
		zeroResultCycles.WithLabelValues(source).Set(0)
		m.mu.Unlock()
		return
	}

	state.zeroCycles++
	cycles := state.zeroCycles
	shouldAlert := !state.alerted && cycles >= m.threshold
	if shouldAlert {
		state.alerted = true
	}
	m.mu.Unlock()

	zeroResultCycles.WithLabelValues(source).Set(float64(cycles))

	if shouldAlert && m.webhook != "" {
		if err := m.sendDiscord(ctx, source, cycles); err != nil {
			m.log.Error("failed to send zero-result alert", "source", source, "error", err)
		} else {
			m.log.Warn("zero-result alert sent", "source", source, "cycles", cycles)
		}
	}
}

func (m *Monitor) sendDiscord(ctx context.Context, source string, cycles int) error {
	payload := map[string]any{
		"content": fmt.Sprintf(
			"**Scraper alert** — `%s` returned **0 series** for **%d** consecutive cycles. The adapter layout may be broken.",
			source,
			cycles,
		),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.webhook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook status %d", resp.StatusCode)
	}
	return nil
}
