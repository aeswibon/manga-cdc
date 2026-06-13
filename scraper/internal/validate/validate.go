package validate

import (
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

type Mode int

const (
	Insert Mode = iota
	Update
)

const maxChapterJump = 50.0

type Issue struct {
	Field   string
	Rule    string
	Message string
}

type Result struct {
	OK     bool
	Issues []Issue
}

type RejectedChapter struct {
	Chapter model.Chapter
	Issues  []Issue
}

type ChapterOptions struct {
	LatestChapter float64
	Now           time.Time
}

var garbageTitles = map[string]struct{}{
	"404":           {},
	"403":           {},
	"500":           {},
	"undefined":     {},
	"null":          {},
	"loading...":    {},
	"loading":       {},
	"not found":     {},
	"error":         {},
	"access denied": {},
}

var allowedStatuses = map[string]struct{}{
	"ONGOING":    {},
	"COMPLETED":  {},
	"HIATUS":     {},
	"CANCELLED":  {},
}

func NormalizeSeries(s model.Series) model.Series {
	s.Title = strings.TrimSpace(s.Title)
	s.SourceID = strings.TrimSpace(s.SourceID)
	s.SourceURL = strings.TrimSpace(s.SourceURL)
	s.CoverURL = strings.TrimSpace(s.CoverURL)
	s.Status = strings.TrimSpace(strings.ToUpper(s.Status))
	if s.Status == "" {
		s.Status = "ONGOING"
	}
	return s
}

func Series(s model.Series, mode Mode) Result {
	var issues []Issue

	title := strings.TrimSpace(s.Title)
	if title == "" {
		issues = append(issues, Issue{Field: "title", Rule: "required", Message: "title is required"})
	} else if isGarbageTitle(title) {
		issues = append(issues, Issue{Field: "title", Rule: "garbage", Message: "title looks like a scrape error"})
	}

	if strings.TrimSpace(s.SourceID) == "" {
		issues = append(issues, Issue{Field: "source_id", Rule: "required", Message: "source_id is required"})
	}

	sourceURL := strings.TrimSpace(s.SourceURL)
	if sourceURL == "" {
		issues = append(issues, Issue{Field: "source_url", Rule: "required", Message: "source_url is required"})
	} else if !isHTTPURL(sourceURL) {
		issues = append(issues, Issue{Field: "source_url", Rule: "format", Message: "source_url must be http or https"})
	}

	coverURL := strings.TrimSpace(s.CoverURL)
	if coverURL != "" && !isHTTPURL(coverURL) {
		issues = append(issues, Issue{Field: "cover_url", Rule: "format", Message: "cover_url must be http or https when set"})
	}

	status := strings.TrimSpace(strings.ToUpper(s.Status))
	if status != "" {
		if _, ok := allowedStatuses[status]; !ok {
			issues = append(issues, Issue{Field: "status", Rule: "enum", Message: "status must be ONGOING, COMPLETED, HIATUS, or CANCELLED"})
		}
	} else if mode == Insert {
		issues = append(issues, Issue{Field: "status", Rule: "required", Message: "status is required"})
	}

	_ = mode

	return Result{
		OK:     len(issues) == 0,
		Issues: issues,
	}
}

func MergeSeries(existing, scraped model.Series) model.Series {
	merged := scraped
	merged.ID = existing.ID

	if strings.TrimSpace(merged.CoverURL) == "" {
		merged.CoverURL = existing.CoverURL
	}
	if strings.TrimSpace(merged.Description) == "" {
		merged.Description = existing.Description
	}
	if strings.TrimSpace(merged.Author) == "" {
		merged.Author = existing.Author
	}
	if strings.TrimSpace(merged.Artist) == "" {
		merged.Artist = existing.Artist
	}
	if strings.TrimSpace(merged.Status) == "" {
		merged.Status = existing.Status
	}
	merged.IsActive = existing.IsActive
	return merged
}

func Chapter(ch model.Chapter, opts ChapterOptions) Result {
	var issues []Issue

	if math.IsNaN(ch.Number) || ch.Number <= 0 {
		issues = append(issues, Issue{Field: "chapter_num", Rule: "range", Message: "chapter_num must be greater than zero"})
	}

	chapterURL := strings.TrimSpace(ch.URL)
	if chapterURL == "" {
		issues = append(issues, Issue{Field: "url", Rule: "required", Message: "url is required"})
	} else if !isHTTPURL(chapterURL) {
		issues = append(issues, Issue{Field: "url", Rule: "format", Message: "url must be http or https"})
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	if !ch.ReleaseDate.IsZero() && ch.ReleaseDate.After(now.Add(24*time.Hour)) {
		issues = append(issues, Issue{Field: "release_date", Rule: "future", Message: "release_date is too far in the future"})
	}

	if opts.LatestChapter > 0 && ch.Number > opts.LatestChapter+maxChapterJump {
		issues = append(issues, Issue{
			Field:   "chapter_num",
			Rule:    "jump",
			Message: "chapter_num jumps too far ahead of the series latest chapter",
		})
	}

	return Result{
		OK:     len(issues) == 0,
		Issues: issues,
	}
}

func FilterChapters(chapters []model.Chapter, opts ChapterOptions) (good []model.Chapter, rejected []RejectedChapter) {
	seen := make(map[float64]struct{}, len(chapters))

	for _, ch := range chapters {
		result := Chapter(ch, opts)
		if !result.OK {
			rejected = append(rejected, RejectedChapter{Chapter: ch, Issues: result.Issues})
			continue
		}
		if _, dup := seen[ch.Number]; dup {
			rejected = append(rejected, RejectedChapter{
				Chapter: ch,
				Issues:  []Issue{{Field: "chapter_num", Rule: "duplicate", Message: "duplicate chapter number in scrape batch"}},
			})
			continue
		}
		seen[ch.Number] = struct{}{}
		good = append(good, ch)
	}

	return good, rejected
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	default:
		return false
	}
}

func isGarbageTitle(title string) bool {
	_, ok := garbageTitles[strings.ToLower(strings.TrimSpace(title))]
	return ok
}
