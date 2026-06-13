package model

import (
	"time"
)

type Series struct {
	ID            string    `json:"id"`
	SourceID      string    `json:"source_id"`
	Title         string    `json:"title"`
	AltTitles     []string  `json:"alt_titles,omitempty"`
	Author        string    `json:"author,omitempty"`
	Artist        string    `json:"artist,omitempty"`
	Description   string    `json:"description,omitempty"`
	CoverURL      string    `json:"cover_url,omitempty"`
	Status        string    `json:"status,omitempty"`
	SourceURL     string    `json:"source_url"`
	LatestChapter float64   `json:"latest_chapter"`
	LastChecked   time.Time `json:"last_checked,omitempty"`
	IsActive      bool      `json:"is_active"`
}

type Chapter struct {
	ID          string    `json:"id"`
	SeriesID    string    `json:"series_id"`
	SeriesTitle string    `json:"series_title,omitempty"`
	Number      float64   `json:"chapter_num"`
	Title       string    `json:"title,omitempty"`
	URL         string    `json:"url"`
	ReleaseDate time.Time `json:"release_date,omitempty"`
	IsNew       bool      `json:"is_new"`
}
