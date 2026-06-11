package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	segkafka "github.com/segmentio/kafka-go"
)

type mockWriter struct {
	lastMsg *segkafka.Message
	err     error
}

func (m *mockWriter) WriteMessages(ctx context.Context, msgs ...segkafka.Message) error {
	if m.err != nil {
		return m.err
	}
	if len(msgs) > 0 {
		m.lastMsg = &msgs[0]
	}
	return nil
}

func (m *mockWriter) Close() error { return nil }

func TestPublishChapterEvent_Success(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{writer: mw, topic: "test-topic"}

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

	if mw.lastMsg == nil {
		t.Fatal("expected a message to be written")
	}

	var payload map[string]any
	if err := json.Unmarshal(mw.lastMsg.Value, &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	op, ok := payload["op"].(string)
	if !ok || op != "c" {
		t.Fatalf("expected op='c', got %v", payload["op"])
	}

	after, ok := payload["after"].(map[string]any)
	if !ok {
		t.Fatal("missing 'after' field")
	}
	if after["id"] != "ch-1" {
		t.Fatalf("expected after.id=ch-1, got %v", after["id"])
	}
	if after["chapter_num"] != float64(1) {
		t.Fatalf("expected after.chapter_num=1, got %v", after["chapter_num"])
	}
	if after["series_id"] != "s-1" {
		t.Fatalf("expected after.series_id=s-1, got %v", after["series_id"])
	}
	if after["url"] != "https://example.com/ch-1" {
		t.Fatalf("expected after.url matches")
	}
}

func TestPublishChapterEvent_WriterError(t *testing.T) {
	mw := &mockWriter{err: errors.New("kafka down")}
	p := &Producer{writer: mw, topic: "test-topic"}

	err := p.PublishChapterEvent(context.Background(), model.Chapter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
