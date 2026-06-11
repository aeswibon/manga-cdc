package qstash

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

type mockHTTPClient struct {
	lastReq *http.Request
	resp    *http.Response
	err     error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.lastReq = req
	return m.resp, nil
}

func TestPublishChapterEvent_Success(t *testing.T) {
	mock := &mockHTTPClient{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("ok"))),
		},
	}
	p := &Publisher{client: mock, token: "test-token", destination: "https://example.com/webhook", apiURL: "https://qstash.upstash.com/v1/publish/"}

	chapter := model.Chapter{
		ID:       "ch-1",
		SeriesID: "s-1",
		Number:   1,
		Title:    "Test Chapter",
		URL:      "https://example.com/ch-1",
		IsNew:    true,
	}

	err := p.PublishChapterEvent(context.Background(), chapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastReq == nil {
		t.Fatal("expected a request to be made")
	}

	if v := mock.lastReq.Header.Get("Authorization"); v != "Bearer test-token" {
		t.Fatalf("expected Authorization=Bearer test-token, got %q", v)
	}

	if v := mock.lastReq.Header.Get("Upstash-Destination"); v != "https://example.com/webhook" {
		t.Fatalf("expected Upstash-Destination=https://example.com/webhook, got %q", v)
	}

	if v := mock.lastReq.Header.Get("Content-Type"); v != "application/json" {
		t.Fatalf("expected Content-Type=application/json, got %q", v)
	}

	body, _ := io.ReadAll(mock.lastReq.Body)
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	op, ok := payload["op"].(string)
	if !ok || op != "c" {
		t.Fatalf("expected op='c', got %v", payload["op"])
	}
}

func TestPublishChapterEvent_HTTPError(t *testing.T) {
	mock := &mockHTTPClient{
		resp: &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewReader([]byte("bad request"))),
		},
	}
	p := &Publisher{client: mock, token: "test-token", destination: "https://example.com/webhook", apiURL: "https://qstash.upstash.com/v1/publish/"}

	err := p.PublishChapterEvent(context.Background(), model.Chapter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPublishChapterEvent_NetworkError(t *testing.T) {
	mock := &mockHTTPClient{err: errors.New("network error")}
	p := &Publisher{client: mock, token: "test-token", destination: "https://example.com/webhook", apiURL: "https://qstash.upstash.com/v1/publish/"}

	err := p.PublishChapterEvent(context.Background(), model.Chapter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
