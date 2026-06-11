package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

func TestHandler_healthz(t *testing.T) {
	mux := http.NewServeMux()
	New(nil, nil, false).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandler_readyz_databaseFailure(t *testing.T) {
	mux := http.NewServeMux()
	New(stubPinger{err: errors.New("db down")}, nil, false).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHandler_readyz_ok(t *testing.T) {
	mux := http.NewServeMux()
	New(stubPinger{}, stubPinger{}, true).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
