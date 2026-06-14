package watchlist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var knownSources = map[string]struct{}{
	"mangadex":   {},
	"mangafire":  {},
	"asurascans": {},
	"mangaplus":  {},
	"mangatown":  {},
	"mangapill":  {},
}

var remoteURLValidator = ValidateRemoteURL

type File struct {
	Series []Entry `yaml:"series"`
}

type Entry struct {
	Source        string             `yaml:"source"`
	SourceID      string             `yaml:"source_id"`
	Title         string             `yaml:"title"`
	SourceURL     string             `yaml:"source_url"`
	CoverURL      string             `yaml:"cover_url,omitempty"`
	Status        string             `yaml:"status,omitempty"`
	Notifications *NotificationPrefs `yaml:"notifications,omitempty"`
}

// NotificationPrefs controls per-series notification behavior (v0.5+).
type NotificationPrefs struct {
	PreferredGroups []string `yaml:"preferred_groups,omitempty"`
	BlockedGroups   []string `yaml:"blocked_groups,omitempty"`
	NotifyEvery     int      `yaml:"notify_every,omitempty"`
	BlockEarlyWeek  bool     `yaml:"block_early_week,omitempty"`
}

// NotificationPrefsJSON returns JSON for DB storage (empty object when unset).
func (e Entry) NotificationPrefsJSON() json.RawMessage {
	if e.Notifications == nil {
		return json.RawMessage("{}")
	}
	b, err := json.Marshal(e.Notifications)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

func ValidateSource(source string) error {
	source = strings.ToLower(strings.TrimSpace(source))
	if _, ok := knownSources[source]; !ok {
		return fmt.Errorf("unknown source %q", source)
	}
	return nil
}

func NamespacedSourceID(source, sourceID string) string {
	return strings.ToLower(strings.TrimSpace(source)) + ":" + strings.TrimSpace(sourceID)
}

func ParseRawSourceID(namespaced string) (source, rawID string, err error) {
	idx := strings.Index(namespaced, ":")
	if idx <= 0 || idx >= len(namespaced)-1 {
		return "", "", fmt.Errorf("invalid namespaced source_id %q", namespaced)
	}
	source = namespaced[:idx]
	rawID = namespaced[idx+1:]
	if err := ValidateSource(source); err != nil {
		return "", "", fmt.Errorf("parse namespaced source_id: %w", err)
	}
	return source, rawID, nil
}

func LoadFromFile(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read watchlist file: %w", err)
	}
	return parse(data)
}

func LoadFromURL(ctx context.Context, url string) ([]Entry, error) {
	if err := remoteURLValidator(url); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create watchlist request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch watchlist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch watchlist: unexpected status %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read watchlist body: %w", err)
	}
	return parse(data)
}

func parse(data []byte) ([]Entry, error) {
	var entries []Entry
	if err := yaml.Unmarshal(data, &entries); err == nil && len(entries) > 0 {
		return validateEntries(entries)
	}

	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse watchlist yaml: %w", err)
	}
	return validateEntries(file.Series)
}

func validateEntries(entries []Entry) ([]Entry, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("watchlist contains no entries")
	}

	for i, entry := range entries {
		if err := validateEntry(entry); err != nil {
			return nil, fmt.Errorf("series[%d]: %w", i, err)
		}
	}
	return entries, nil
}

func validateEntry(entry Entry) error {
	if err := ValidateSource(entry.Source); err != nil {
		return err
	}
	if strings.TrimSpace(entry.SourceID) == "" {
		return fmt.Errorf("source_id is required")
	}
	if strings.TrimSpace(entry.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(entry.SourceURL) == "" {
		return fmt.Errorf("source_url is required")
	}
	if entry.Notifications != nil {
		if err := validateNotificationPrefs(entry.Notifications); err != nil {
			return err
		}
	}
	return nil
}

func validateNotificationPrefs(prefs *NotificationPrefs) error {
	if prefs.NotifyEvery < 0 {
		return fmt.Errorf("notifications.notify_every must be >= 0")
	}
	for _, group := range prefs.PreferredGroups {
		if strings.TrimSpace(group) == "" {
			return fmt.Errorf("notifications.preferred_groups must not contain empty strings")
		}
	}
	for _, group := range prefs.BlockedGroups {
		if strings.TrimSpace(group) == "" {
			return fmt.Errorf("notifications.blocked_groups must not contain empty strings")
		}
	}
	return nil
}
