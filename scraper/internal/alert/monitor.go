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

var rejectRateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "scraper_reject_rate",
	Help: "Most recent validation reject rate per source (0-1)",
}, []string{"source"})

type Config struct {
	ZeroResultThreshold int
	RejectRateThreshold float64
	RejectRateMinSample int
}

type Monitor struct {
	log                 *slog.Logger
	webhook             string
	zeroThreshold       int
	rejectRateThreshold float64
	rejectRateMinSample int
	client              *http.Client

	mu      sync.Mutex
	sources map[string]*sourceState
}

type sourceState struct {
	zeroCycles    int
	zeroAlerted   bool
	rejectAlerted bool
}

func New(log *slog.Logger, webhookURL string, cfg Config) *Monitor {
	if cfg.ZeroResultThreshold < 1 {
		cfg.ZeroResultThreshold = 3
	}
	if cfg.RejectRateThreshold <= 0 || cfg.RejectRateThreshold > 1 {
		cfg.RejectRateThreshold = 0.5
	}
	if cfg.RejectRateMinSample < 1 {
		cfg.RejectRateMinSample = 5
	}
	return &Monitor{
		log:                 log,
		webhook:             webhookURL,
		zeroThreshold:       cfg.ZeroResultThreshold,
		rejectRateThreshold: cfg.RejectRateThreshold,
		rejectRateMinSample: cfg.RejectRateMinSample,
		client:              &http.Client{Timeout: 10 * time.Second},
		sources:             make(map[string]*sourceState),
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
		state.zeroAlerted = false
		zeroResultCycles.WithLabelValues(source).Set(0)
		m.mu.Unlock()
		return
	}

	state.zeroCycles++
	cycles := state.zeroCycles
	shouldAlert := !state.zeroAlerted && cycles >= m.zeroThreshold
	if shouldAlert {
		state.zeroAlerted = true
	}
	m.mu.Unlock()

	zeroResultCycles.WithLabelValues(source).Set(float64(cycles))

	if shouldAlert && m.webhook != "" {
		if err := m.sendDiscord(ctx, fmt.Sprintf(
			"**Scraper alert** — `%s` returned **0 series** for **%d** consecutive cycles. The adapter layout may be broken.",
			source,
			cycles,
		)); err != nil {
			m.log.Error("failed to send zero-result alert", "source", source, "error", err)
		} else {
			m.log.Warn("zero-result alert sent", "source", source, "cycles", cycles)
		}
	}
}

func (m *Monitor) RecordValidation(ctx context.Context, source string, fetched, rejected int) {
	if fetched <= 0 {
		rejectRateGauge.WithLabelValues(source).Set(0)
		return
	}

	rate := float64(rejected) / float64(fetched)
	rejectRateGauge.WithLabelValues(source).Set(rate)

	m.mu.Lock()
	state := m.sources[source]
	if state == nil {
		state = &sourceState{}
		m.sources[source] = state
	}

	if fetched < m.rejectRateMinSample || rate < m.rejectRateThreshold {
		state.rejectAlerted = false
		m.mu.Unlock()
		return
	}

	shouldAlert := !state.rejectAlerted
	if shouldAlert {
		state.rejectAlerted = true
	}
	m.mu.Unlock()

	if shouldAlert && m.webhook != "" {
		pct := int(rate * 100)
		if err := m.sendDiscord(ctx, fmt.Sprintf(
			"**Scraper alert** — `%s` rejected **%d/%d** series (**%d%%**) this cycle. Validation guardrails may be filtering a broken scrape.",
			source,
			rejected,
			fetched,
			pct,
		)); err != nil {
			m.log.Error("failed to send reject-rate alert", "source", source, "error", err)
		} else {
			m.log.Warn("reject-rate alert sent", "source", source, "rejected", rejected, "fetched", fetched, "rate", rate)
		}
	}
}

func (m *Monitor) sendDiscord(ctx context.Context, content string) error {
	payload := map[string]any{"content": content}
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
