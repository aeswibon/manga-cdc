package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const anilistGraphQLURL = "https://graphql.anilist.co"

type Resolver struct {
	client *http.Client
	log    *slog.Logger
}

func NewResolver() *Resolver {
	return &Resolver{
		client: &http.Client{Timeout: 15 * time.Second},
		log:    slog.Default().With("component", "metadata_resolver"),
	}
}

type Metadata struct {
	AniListID      int
	MalID          *int
	CanonicalTitle string
}

type anilistQuery struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type anilistResponse struct {
	Data struct {
		Media struct {
			ID    int `json:"id"`
			IDMal int `json:"idMal"`
			Title struct {
				Romaji  string `json:"romaji"`
				English string `json:"english"`
				Native  string `json:"native"`
			} `json:"title"`
		} `json:"Media"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (r *Resolver) Resolve(ctx context.Context, title string, altTitles []string) (*Metadata, error) {
	// Simple search heuristic: try the main title first
	q := `
	query ($search: String) {
		Media(search: $search, type: MANGA) {
			id
			idMal
			title {
				romaji
				english
				native
			}
		}
	}
	`

	// Clean up title
	searchTerm := strings.TrimSpace(title)

	payload := anilistQuery{
		Query: q,
		Variables: map[string]interface{}{
			"search": searchTerm,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistGraphQLURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			// No results found
			return nil, nil
		}
		// AniList often returns 429 for rate limits
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			return nil, fmt.Errorf("anilist rate limit exceeded, retry after %s", retryAfter)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var res anilistResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(res.Errors) > 0 {
		if strings.Contains(res.Errors[0].Message, "Not Found") {
			return nil, nil
		}
		return nil, fmt.Errorf("anilist error: %s", res.Errors[0].Message)
	}

	media := res.Data.Media
	if media.ID == 0 {
		return nil, nil
	}

	// Determine best canonical title
	canonicalTitle := media.Title.English
	if canonicalTitle == "" {
		canonicalTitle = media.Title.Romaji
	}
	if canonicalTitle == "" {
		canonicalTitle = media.Title.Native
	}
	if canonicalTitle == "" {
		canonicalTitle = searchTerm // Fallback
	}

	var malID *int
	if media.IDMal != 0 {
		malID = &media.IDMal
	}

	return &Metadata{
		AniListID:      media.ID,
		MalID:          malID,
		CanonicalTitle: canonicalTitle,
	}, nil
}
