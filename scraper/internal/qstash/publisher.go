package qstash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Publisher struct {
	client       HTTPClient
	token        string
	destination  string
	apiURL       string
}

type chapterEvent struct {
	Op    string       `json:"op"`
	After chapterData  `json:"after"`
}

type chapterData struct {
	ID          string  `json:"id"`
	SeriesID    string  `json:"series_id"`
	ChapterNum   float64 `json:"chapter_num"`
	Title       string  `json:"title,omitempty"`
	URL         string  `json:"url"`
	IsNew       bool    `json:"is_new"`
}

func NewPublisher(token, destination string) *Publisher {
	return &Publisher{
		client:      http.DefaultClient,
		token:       token,
		destination: destination,
		apiURL:      "https://qstash.upstash.com/v1/publish/",
	}
}

func (p *Publisher) PublishChapterEvent(ctx context.Context, chapter model.Chapter) error {
	event := chapterEvent{
		Op: "c",
		After: chapterData{
			ID:         chapter.ID,
			SeriesID:   chapter.SeriesID,
			ChapterNum:  chapter.Number,
			Title:      chapter.Title,
			URL:        chapter.URL,
			IsNew:      true,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal chapter event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Upstash-Destination", p.destination)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("publish to qstash: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qstash returned status %d", resp.StatusCode)
	}

	return nil
}
