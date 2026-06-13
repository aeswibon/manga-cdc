package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	segkafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...segkafka.Message) error
	Close() error
}

type Producer struct {
	writer  Writer
	topic   string
	brokers []string
}

func NewProducer(brokers, topic, username, password string) (*Producer, error) {
	w := &segkafka.Writer{
		Addr:         segkafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		BatchTimeout: 50 * time.Millisecond,
		BatchSize:    1,
	}

	if username != "" && password != "" {
		mechanism, err := scram.Mechanism(scram.SHA256, username, password)
		if err != nil {
			return nil, fmt.Errorf("create SCRAM mechanism: %w", err)
		}
		w.Transport = &segkafka.Transport{
			SASL: mechanism,
			TLS:  &tls.Config{MinVersion: tls.VersionTLS12},
		}
	}

	return &Producer{
		writer:  w,
		topic:   topic,
		brokers: strings.Split(brokers, ","),
	}, nil
}

type chapterEvent struct {
	Op    string       `json:"op"`
	After chapterData  `json:"after"`
}

type chapterData struct {
	ID          string  `json:"id"`
	SeriesID    string  `json:"series_id"`
	SeriesTitle string  `json:"series_title,omitempty"`
	ChapterNum  float64 `json:"chapter_num"`
	Title       string  `json:"title,omitempty"`
	URL         string  `json:"url"`
	IsNew       bool    `json:"is_new"`
}

func (p *Producer) PublishChapterEvent(ctx context.Context, chapter model.Chapter) error {
	event := chapterEvent{
		Op: "c",
		After: chapterData{
			ID:          chapter.ID,
			SeriesID:    chapter.SeriesID,
			SeriesTitle: chapter.SeriesTitle,
			ChapterNum:  chapter.Number,
			Title:       chapter.Title,
			URL:         chapter.URL,
			IsNew:       true,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal chapter event: %w", err)
	}

	msg := segkafka.Message{
		Key:   []byte(chapter.ID),
		Value: data,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write chapter event: %w", err)
	}

	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func (p *Producer) Ping(ctx context.Context) error {
	if len(p.brokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", strings.TrimSpace(p.brokers[0]))
	if err != nil {
		return fmt.Errorf("dial kafka broker: %w", err)
	}
	return conn.Close()
}
