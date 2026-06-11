//go:build integration

package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestDB(t *testing.T) *DB {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = "postgres://mangacdc:mangacdc@localhost:5432/mangacdc?sslmode=disable"
	}

	db, err := New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test DB: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func migrate(t *testing.T, dsn string) {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect for migration: %v", err)
	}
	defer pool.Close()

	migration, err := os.ReadFile("../../db/migrations/001_initial_schema.sql")
	if err != nil {
		// try relative to test file
		migration, err = os.ReadFile("../../../db/migrations/001_initial_schema.sql")
		if err != nil {
			t.Fatalf("read migration: %v", err)
		}
	}

	statements := strings.Split(string(migration), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(context.Background(), stmt); err != nil {
			// ignore "already exists" errors for idempotency
			if !strings.Contains(err.Error(), "already exists") {
				t.Fatalf("migration statement: %v\nSQL: %s", err, stmt)
			}
		}
	}
}

func cleanDB(t *testing.T, dsn string) {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect for cleanup: %v", err)
	}
	defer pool.Close()

	pool.Exec(context.Background(), "DELETE FROM notification_logs")
	pool.Exec(context.Background(), "DELETE FROM chapters")
	pool.Exec(context.Background(), "DELETE FROM manga_series")
}

func dsn() string {
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = "postgres://mangacdc:mangacdc@localhost:5432/mangacdc?sslmode=disable"
	}
	return dsn
}

func TestIntegration_UpsertSeries(t *testing.T) {
	dsn := dsn()
	migrate(t, dsn)
	cleanDB(t, dsn)
	db := getTestDB(t)

	series := model.Series{
		SourceID:  "test-source-1",
		Title:     "Test Series",
		CoverURL:  "https://example.com/cover.jpg",
		SourceURL: "https://example.com/test-series",
		Status:    "ONGOING",
		IsActive:  true,
	}

	id, err := db.UpsertSeries(context.Background(), series)
	if err != nil {
		t.Fatalf("UpsertSeries: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	got, err := db.GetSeriesBySourceID(context.Background(), "test-source-1")
	if err != nil {
		t.Fatalf("GetSeriesBySourceID: %v", err)
	}
	if got.Title != "Test Series" {
		t.Errorf("title = %q, want %q", got.Title, "Test Series")
	}
	if got.Status != "ONGOING" {
		t.Errorf("status = %q, want %q", got.Status, "ONGOING")
	}
}

func TestIntegration_UpsertSeries_UpdateExisting(t *testing.T) {
	dsn := dsn()
	migrate(t, dsn)
	cleanDB(t, dsn)
	db := getTestDB(t)

	series := model.Series{
		SourceID:  "test-source-2",
		Title:     "Original Title",
		SourceURL: "https://example.com/original",
		Status:    "ONGOING",
		IsActive:  true,
	}

	id1, err := db.UpsertSeries(context.Background(), series)
	if err != nil {
		t.Fatalf("first UpsertSeries: %v", err)
	}

	series.Title = "Updated Title"
	series.Status = "COMPLETED"
	id2, err := db.UpsertSeries(context.Background(), series)
	if err != nil {
		t.Fatalf("second UpsertSeries: %v", err)
	}

	if id1 != id2 {
		t.Errorf("expected same ID after upsert: %s vs %s", id1, id2)
	}

	got, _ := db.GetSeriesBySourceID(context.Background(), "test-source-2")
	if got.Title != "Updated Title" {
		t.Errorf("title = %q, want %q", got.Title, "Updated Title")
	}
	if got.Status != "COMPLETED" {
		t.Errorf("status = %q, want %q", got.Status, "COMPLETED")
	}
}

func TestIntegration_InsertChapter(t *testing.T) {
	dsn := dsn()
	migrate(t, dsn)
	cleanDB(t, dsn)
	db := getTestDB(t)

	series := model.Series{
		SourceID:  "chapter-test-series",
		Title:     "Chapter Test",
		SourceURL: "https://example.com/chapter-test",
		Status:    "ONGOING",
		IsActive:  true,
	}
	seriesID, err := db.UpsertSeries(context.Background(), series)
	if err != nil {
		t.Fatalf("UpsertSeries: %v", err)
	}

	id, err := db.InsertChapter(context.Background(), seriesID, model.Chapter{
		Number: 1,
		Title:  "Chapter 1",
		URL:    "https://example.com/chapter-test/ch-1",
		IsNew:  true,
	})
	if err != nil {
		t.Fatalf("InsertChapter: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	_, err = db.InsertChapter(context.Background(), seriesID, model.Chapter{
		Number: 1,
		Title:  "Chapter 1 again",
		URL:    "https://example.com/dup",
		IsNew:  true,
	})
	if err != nil {
		t.Fatalf("duplicate InsertChapter should no-op: %v", err)
	}

	chapters, err := db.GetNewChapters(context.Background())
	if err != nil {
		t.Fatalf("GetNewChapters: %v", err)
	}
	if len(chapters) != 1 {
		t.Fatalf("expected 1 new chapter, got %d", len(chapters))
	}
}

func TestIntegration_MarkChapterNotified(t *testing.T) {
	dsn := dsn()
	migrate(t, dsn)
	cleanDB(t, dsn)
	db := getTestDB(t)

	series := model.Series{
		SourceID:  "notify-test",
		Title:     "Notify Test",
		SourceURL: "https://example.com/notify-test",
		Status:    "ONGOING",
		IsActive:  true,
	}
	seriesID, _ := db.UpsertSeries(context.Background(), series)

	chID, _ := db.InsertChapter(context.Background(), seriesID, model.Chapter{
		Number: 1,
		Title:  "Chapter 1",
		URL:    "https://example.com/ch-1",
		IsNew:  true,
	})

	err := db.MarkChapterNotified(context.Background(), chID)
	if err != nil {
		t.Fatalf("MarkChapterNotified: %v", err)
	}

	chapters, _ := db.GetNewChapters(context.Background())
	if len(chapters) != 0 {
		t.Errorf("expected 0 new chapters after marking notified, got %d", len(chapters))
	}
}

func TestIntegration_InsertNotificationLog(t *testing.T) {
	dsn := dsn()
	migrate(t, dsn)
	cleanDB(t, dsn)
	db := getTestDB(t)

	series := model.Series{
		SourceID:  "log-test",
		Title:     "Log Test",
		SourceURL: "https://example.com/log-test",
		Status:    "ONGOING",
		IsActive:  true,
	}
	seriesID, _ := db.UpsertSeries(context.Background(), series)
	chID, _ := db.InsertChapter(context.Background(), seriesID, model.Chapter{
		Number: 1,
		Title:  "Chapter 1",
		URL:    "https://example.com/ch-1",
		IsNew:  true,
	})

	err := db.InsertNotificationLog(context.Background(), chID, "SENT", "discord", "")
	if err != nil {
		t.Fatalf("InsertNotificationLog: %v", err)
	}

	err = db.InsertNotificationLog(context.Background(), chID, "FAILED", "slack", "timeout")
	if err != nil {
		t.Fatalf("InsertNotificationLog (failed): %v", err)
	}
}

func TestIntegration_GetActiveSeries(t *testing.T) {
	dsn := dsn()
	migrate(t, dsn)
	cleanDB(t, dsn)
	db := getTestDB(t)

	for i := range 3 {
		active := true
		if i == 2 {
			active = false
		}
		_, err := db.UpsertSeries(context.Background(), model.Series{
			SourceID:  "active-test-" + time.Now().String()[:10],
			Title:     "Active Test",
			SourceURL: "https://example.com/active",
			Status:    "ONGOING",
			IsActive:  active,
		})
		if err != nil {
			t.Fatalf("UpsertSeries: %v", err)
		}
	}

	active, err := db.GetActiveSeries(context.Background())
	if err != nil {
		t.Fatalf("GetActiveSeries: %v", err)
	}
	if len(active) == 0 {
		t.Fatal("expected at least 1 active series")
	}
}
