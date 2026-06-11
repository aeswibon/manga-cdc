package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type KafkaPinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db          DBPinger
	kafka       KafkaPinger
	kafkaNeeded atomic.Bool
}

func New(db DBPinger, kafka KafkaPinger, kafkaRequired bool) *Handler {
	h := &Handler{db: db, kafka: kafka}
	if kafkaRequired {
		h.kafkaNeeded.Store(true)
	}
	return h
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{"database": "ok"}

	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["database"] = err.Error()
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "unavailable",
				"checks": checks,
			})
			return
		}
	}

	if h.kafkaNeeded.Load() && h.kafka != nil {
		if err := h.kafka.Ping(ctx); err != nil {
			checks["kafka"] = err.Error()
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "unavailable",
				"checks": checks,
			})
			return
		}
		checks["kafka"] = "ok"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ready",
		"checks": checks,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
