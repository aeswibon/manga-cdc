package validate

import (
	"testing"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

func TestSeries_Insert(t *testing.T) {
	tests := []struct {
		name     string
		series   model.Series
		wantOK   bool
		wantRule string
	}{
		{
			name: "valid insert",
			series: model.Series{
				SourceID:  "abc",
				Title:     "One Piece",
				SourceURL: "https://mangadex.org/title/abc",
				CoverURL:  "https://mangadex.org/covers/abc/cover.jpg",
				Status:    "ONGOING",
			},
			wantOK: true,
		},
		{
			name: "missing title",
			series: model.Series{
				SourceID:  "abc",
				SourceURL: "https://example.com/title/abc",
				CoverURL:  "https://example.com/cover.jpg",
				Status:    "ONGOING",
			},
			wantOK:   false,
			wantRule: "required",
		},
		{
			name: "missing cover on insert allowed",
			series: model.Series{
				SourceID:  "abc",
				Title:     "One Piece",
				SourceURL: "https://example.com/title/abc",
				Status:    "ONGOING",
			},
			wantOK: true,
		},
		{
			name: "garbage title",
			series: model.Series{
				SourceID:  "abc",
				Title:     "404",
				SourceURL: "https://example.com/title/abc",
				CoverURL:  "https://example.com/cover.jpg",
				Status:    "ONGOING",
			},
			wantOK:   false,
			wantRule: "garbage",
		},
		{
			name: "invalid source url",
			series: model.Series{
				SourceID:  "abc",
				Title:     "One Piece",
				SourceURL: "not-a-url",
				CoverURL:  "https://example.com/cover.jpg",
				Status:    "ONGOING",
			},
			wantOK:   false,
			wantRule: "format",
		},
		{
			name: "invalid status",
			series: model.Series{
				SourceID:  "abc",
				Title:     "One Piece",
				SourceURL: "https://example.com/title/abc",
				Status:    "RUNNING",
			},
			wantOK:   false,
			wantRule: "enum",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Series(tc.series, Insert)
			if result.OK != tc.wantOK {
				t.Fatalf("Series() OK = %v, want %v; issues = %+v", result.OK, tc.wantOK, result.Issues)
			}
			if !tc.wantOK && tc.wantRule != "" {
				found := false
				for _, issue := range result.Issues {
					if issue.Rule == tc.wantRule {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected rule %q in issues %+v", tc.wantRule, result.Issues)
				}
			}
		})
	}
}

func TestNormalizeSeries_defaultsStatus(t *testing.T) {
	normalized := NormalizeSeries(model.Series{Title: "One Piece"})
	if normalized.Status != "ONGOING" {
		t.Fatalf("expected ONGOING, got %q", normalized.Status)
	}
}

func TestSeries_UpdateAllowsMissingCoverAfterMerge(t *testing.T) {
	existing := model.Series{
		ID:        "id-1",
		SourceID:  "abc",
		Title:     "One Piece",
		SourceURL: "https://example.com/title/abc",
		CoverURL:  "https://example.com/cover.jpg",
		Status:    "ONGOING",
		IsActive:  true,
	}
	scraped := model.Series{
		SourceID:  "abc",
		Title:     "One Piece",
		SourceURL: "https://example.com/title/abc",
		Status:    "ONGOING",
	}
	merged := MergeSeries(existing, scraped)

	result := Series(merged, Update)
	if !result.OK {
		t.Fatalf("expected update to pass after merge, got issues %+v", result.Issues)
	}
	if merged.CoverURL != existing.CoverURL {
		t.Fatalf("expected merged cover %q, got %q", existing.CoverURL, merged.CoverURL)
	}
}

func TestMergeSeries_PreservesInactive(t *testing.T) {
	existing := model.Series{ID: "id-1", IsActive: false, CoverURL: "https://example.com/cover.jpg"}
	scraped := model.Series{IsActive: true}
	merged := MergeSeries(existing, scraped)
	if merged.IsActive {
		t.Fatal("expected is_active to remain false")
	}
}

func TestChapter(t *testing.T) {
	tests := []struct {
		name     string
		chapter  model.Chapter
		opts     ChapterOptions
		wantOK   bool
		wantRule string
	}{
		{
			name:    "valid",
			chapter: model.Chapter{Number: 1, URL: "https://example.com/ch/1"},
			wantOK:  true,
		},
		{
			name:    "zero chapter",
			chapter: model.Chapter{Number: 0, URL: "https://example.com/ch/0"},
			wantOK:  false,
		},
		{
			name:    "missing url",
			chapter: model.Chapter{Number: 1},
			wantOK:  false,
		},
		{
			name: "future release date",
			chapter: model.Chapter{
				Number:      1,
				URL:         "https://example.com/ch/1",
				ReleaseDate: time.Now().Add(48 * time.Hour),
			},
			wantOK:   false,
			wantRule: "future",
		},
		{
			name: "chapter jump",
			chapter: model.Chapter{
				Number: 200,
				URL:    "https://example.com/ch/200",
			},
			opts:     ChapterOptions{LatestChapter: 10},
			wantOK:   false,
			wantRule: "jump",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Chapter(tc.chapter, tc.opts)
			if result.OK != tc.wantOK {
				t.Fatalf("Chapter() OK = %v, want %v; issues = %+v", result.OK, tc.wantOK, result.Issues)
			}
			if !tc.wantOK && tc.wantRule != "" {
				found := false
				for _, issue := range result.Issues {
					if issue.Rule == tc.wantRule {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected rule %q in issues %+v", tc.wantRule, result.Issues)
				}
			}
		})
	}
}

func TestFilterChapters_DedupesBatch(t *testing.T) {
	chapters := []model.Chapter{
		{Number: 1, URL: "https://example.com/ch/1"},
		{Number: 1, URL: "https://example.com/ch/1-dup"},
		{Number: 2, URL: "https://example.com/ch/2"},
	}
	good, rejected := FilterChapters(chapters, ChapterOptions{})
	if len(good) != 2 {
		t.Fatalf("expected 2 good chapters, got %d", len(good))
	}
	if len(rejected) != 1 {
		t.Fatalf("expected 1 rejected chapter, got %d", len(rejected))
	}
	if rejected[0].Issues[0].Rule != "duplicate" {
		t.Fatalf("expected duplicate rejection, got %+v", rejected[0].Issues)
	}
}
