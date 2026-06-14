package watchlist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNamespacedSourceID(t *testing.T) {
	got := NamespacedSourceID("MangaDex", " abc-123 ")
	if got != "mangadex:abc-123" {
		t.Fatalf("got %q, want mangadex:abc-123", got)
	}
}

func TestParseRawSourceID(t *testing.T) {
	source, rawID, err := ParseRawSourceID("mangaplus:100020")
	if err != nil {
		t.Fatal(err)
	}
	if source != "mangaplus" || rawID != "100020" {
		t.Fatalf("got (%q, %q)", source, rawID)
	}

	_, _, err = ParseRawSourceID("invalid")
	if err == nil {
		t.Fatal("expected error for invalid namespaced id")
	}

	_, _, err = ParseRawSourceID("unknown:abc")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestValidateSource(t *testing.T) {
	if err := ValidateSource("mangadex"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSource("not-a-source"); err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestLoadFromFile(t *testing.T) {
	path := filepath.Join("testdata", "watchlist.yaml")
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	content := `series:
  - source: mangadex
    source_id: "abc-123"
    title: "Test Manga"
    source_url: "https://mangadex.org/title/abc-123"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(path) })

	entries, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Title != "Test Manga" {
		t.Fatalf("unexpected title %q", entries[0].Title)
	}
}

func TestLoadFromFile_rootListFormat(t *testing.T) {
	path := filepath.Join("testdata", "root-watchlist.yaml")
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	content := `- source: mangadex
  source_id: "abc-123"
  title: "Test Manga"
  source_url: "https://mangadex.org/title/abc-123"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(path) })

	entries, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Title != "Test Manga" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestLoadFromFile_rejectsInvalidSource(t *testing.T) {
	path := filepath.Join("testdata", "bad-watchlist.yaml")
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	content := `series:
  - source: unknown
    source_id: "1"
    title: "Bad"
    source_url: "https://example.com"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(path) })

	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateEntry_notificationPrefs(t *testing.T) {
	entry := Entry{
		Source:    "mangadex",
		SourceID:  "abc",
		Title:     "Test",
		SourceURL: "https://mangadex.org/title/abc",
		Notifications: &NotificationPrefs{
			NotifyEvery:    5,
			PreferredGroups: []string{"Official TL"},
		},
	}
	if err := validateEntry(entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry.Notifications.NotifyEvery = -1
	if err := validateEntry(entry); err == nil {
		t.Fatal("expected error for negative notify_every")
	}
}

func TestNotificationPrefsJSON(t *testing.T) {
	entry := Entry{
		Notifications: &NotificationPrefs{NotifyEvery: 10},
	}
	raw := entry.NotificationPrefsJSON()
	if string(raw) != `{"notify_every":10}` {
		t.Fatalf("unexpected json %s", raw)
	}

	empty := Entry{}.NotificationPrefsJSON()
	if string(empty) != "{}" {
		t.Fatalf("expected empty object, got %s", empty)
	}
}

func TestLoadRepoWatchlist(t *testing.T) {
	path := filepath.Join("..", "..", "..", "data", "watchlist.yaml")
	entries, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 1 {
		t.Fatal("expected at least one entry in repo watchlist")
	}
}

func TestLoadFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte(`series:
  - source: mangaplus
    source_id: "42"
    title: "Remote Manga"
    source_url: "https://mangaplus.shueisha.co.jp/titles/42"
`))
	}))
	defer srv.Close()

	remoteURLValidator = func(string) error { return nil }
	t.Cleanup(func() { remoteURLValidator = ValidateRemoteURL })

	entries, err := LoadFromURL(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].SourceID != "42" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}
