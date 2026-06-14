package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string, maxConns int) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	if maxConns < 1 {
		maxConns = 5
	}
	config.MaxConns = int32(maxConns)
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *DB) UpsertSeries(ctx context.Context, s model.Series) (string, error) {
	var id string
	err := d.pool.QueryRow(ctx, `
		INSERT INTO manga_series (source_id, title, alt_titles, anilist_id, mal_id, canonical_title, author, artist, description,
			cover_url, status, source_url, latest_chapter, last_checked, is_active, notification_prefs)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, COALESCE($16, '{}'::jsonb))
		ON CONFLICT (source_id) DO UPDATE SET
			title = EXCLUDED.title,
			alt_titles = EXCLUDED.alt_titles,
			anilist_id = EXCLUDED.anilist_id,
			mal_id = EXCLUDED.mal_id,
			canonical_title = EXCLUDED.canonical_title,
			author = EXCLUDED.author,
			artist = EXCLUDED.artist,
			description = EXCLUDED.description,
			cover_url = EXCLUDED.cover_url,
			status = EXCLUDED.status,
			latest_chapter = EXCLUDED.latest_chapter,
			last_checked = EXCLUDED.last_checked,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING id
	`, s.SourceID, s.Title, s.AltTitles, s.AniListID, s.MalID, s.CanonicalTitle, s.Author, s.Artist, s.Description,
		s.CoverURL, s.Status, s.SourceURL, s.LatestChapter, time.Now(), s.IsActive, prefsOrEmpty(s.NotificationPrefs)).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("upsert series: %w", err)
	}
	return id, nil
}

func prefsOrEmpty(prefs json.RawMessage) json.RawMessage {
	if len(prefs) == 0 {
		return json.RawMessage("{}")
	}
	return prefs
}

func (d *DB) UpdateSeriesNotificationPrefs(ctx context.Context, sourceID string, prefs json.RawMessage) error {
	_, err := d.pool.Exec(ctx, `
		UPDATE manga_series SET
			notification_prefs = COALESCE($2, '{}'::jsonb),
			updated_at = NOW()
		WHERE source_id = $1
	`, sourceID, prefsOrEmpty(prefs))
	if err != nil {
		return fmt.Errorf("update series notification prefs: %w", err)
	}
	return nil
}

func (d *DB) UpdateSeries(ctx context.Context, s model.Series) error {
	_, err := d.pool.Exec(ctx, `
		UPDATE manga_series SET
			title = $1,
			alt_titles = $2,
			anilist_id = $3,
			mal_id = $4,
			canonical_title = $5,
			author = $6,
			artist = $7,
			description = $8,
			cover_url = $9,
			status = $10,
			source_url = $11,
			latest_chapter = $12,
			last_checked = NOW(),
			is_active = $13,
			updated_at = NOW()
		WHERE id = $14
	`, s.Title, s.AltTitles, s.AniListID, s.MalID, s.CanonicalTitle, s.Author, s.Artist, s.Description,
		s.CoverURL, s.Status, s.SourceURL, s.LatestChapter, s.IsActive, s.ID)

	if err != nil {
		return fmt.Errorf("update series: %w", err)
	}
	return nil
}

func (d *DB) GetSeriesBySourceID(ctx context.Context, sourceID string) (*model.Series, error) {
	var s model.Series
	err := d.pool.QueryRow(ctx, `
		SELECT id, source_id, title, COALESCE(alt_titles, '[]'::jsonb), anilist_id, mal_id, canonical_title, author, artist,
			description, cover_url, status, source_url, latest_chapter, is_active
		FROM manga_series WHERE source_id = $1
	`, sourceID).Scan(
		&s.ID, &s.SourceID, &s.Title, &s.AltTitles, &s.AniListID, &s.MalID, &s.CanonicalTitle, &s.Author, &s.Artist,
		&s.Description, &s.CoverURL, &s.Status, &s.SourceURL, &s.LatestChapter, &s.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get series by source id: %w", err)
	}
	return &s, nil
}

func (d *DB) GetSeriesByTitle(ctx context.Context, title string) (*model.Series, error) {
	var s model.Series
	err := d.pool.QueryRow(ctx, `
		SELECT id, source_id, title, COALESCE(alt_titles, '[]'::jsonb), anilist_id, mal_id, canonical_title, author, artist,
			description, cover_url, status, source_url, latest_chapter, is_active
		FROM manga_series WHERE LOWER(TRIM(title)) = LOWER(TRIM($1))
		LIMIT 1
	`, title).Scan(
		&s.ID, &s.SourceID, &s.Title, &s.AltTitles, &s.AniListID, &s.MalID, &s.CanonicalTitle, &s.Author, &s.Artist,
		&s.Description, &s.CoverURL, &s.Status, &s.SourceURL, &s.LatestChapter, &s.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get series by title: %w", err)
	}
	return &s, nil
}

func (d *DB) InsertChapter(ctx context.Context, seriesID string, ch model.Chapter) (string, error) {
	var id string
	err := d.pool.QueryRow(ctx, `
		INSERT INTO chapters (series_id, chapter_num, title, url, release_date, is_new)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (series_id, chapter_num) DO NOTHING
		RETURNING id
	`, seriesID, ch.Number, ch.Title, ch.URL, ch.ReleaseDate).Scan(&id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("insert chapter: %w", err)
	}
	return id, nil
}

func (d *DB) BulkInsertChapters(ctx context.Context, seriesID string, chapters []model.Chapter) ([]model.Chapter, error) {
	if len(chapters) == 0 {
		return nil, nil
	}

	batch := &pgx.Batch{}
	for _, ch := range chapters {
		batch.Queue(`
			INSERT INTO chapters (series_id, chapter_num, title, url, release_date, is_new)
			VALUES ($1, $2, $3, $4, $5, true)
			ON CONFLICT (series_id, chapter_num) DO NOTHING
			RETURNING id, chapter_num
		`, seriesID, ch.Number, ch.Title, ch.URL, ch.ReleaseDate)
	}

	br := d.pool.SendBatch(ctx, batch)
	defer br.Close()

	byNum := make(map[float64]model.Chapter, len(chapters))
	for _, ch := range chapters {
		byNum[ch.Number] = ch
	}

	var newChapters []model.Chapter
	for i := 0; i < len(chapters); i++ {
		var id string
		var chapterNum float64
		err := br.QueryRow().Scan(&id, &chapterNum)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("scan chapter from batch: %w", err)
		}
		if ref, ok := byNum[chapterNum]; ok {
			ref.ID = id
			ref.SeriesID = seriesID
			ref.IsNew = true
			newChapters = append(newChapters, ref)
		}
	}

	return newChapters, nil
}

func (d *DB) DeleteChaptersForSeries(ctx context.Context, seriesID string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM chapters WHERE series_id = $1`, seriesID)
	if err != nil {
		return fmt.Errorf("delete chapters for series: %w", err)
	}
	return nil
}

func (d *DB) GetActiveSeries(ctx context.Context) ([]model.Series, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, source_id, title, COALESCE(alt_titles, '[]'::jsonb), anilist_id, mal_id, canonical_title, author, artist,
			description, cover_url, status, source_url, latest_chapter, is_active
		FROM manga_series WHERE is_active = true
	`)
	if err != nil {
		return nil, fmt.Errorf("get active series: %w", err)
	}
	defer rows.Close()

	var series []model.Series
	for rows.Next() {
		var s model.Series
		err := rows.Scan(
			&s.ID, &s.SourceID, &s.Title, &s.AltTitles, &s.AniListID, &s.MalID, &s.CanonicalTitle, &s.Author, &s.Artist,
			&s.Description, &s.CoverURL, &s.Status, &s.SourceURL, &s.LatestChapter, &s.IsActive)
		if err != nil {
			return nil, fmt.Errorf("scan series: %w", err)
		}
		series = append(series, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active series: %w", err)
	}
	return series, nil
}

func (d *DB) DeleteSeriesExceptSourceIDs(ctx context.Context, keepSourceIDs []string) (int64, error) {
	if len(keepSourceIDs) == 0 {
		return 0, nil
	}

	tag, err := d.pool.Exec(ctx, `
		DELETE FROM manga_series
		WHERE NOT (source_id = ANY($1))
	`, keepSourceIDs)
	if err != nil {
		return 0, fmt.Errorf("delete series not in watchlist: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (d *DB) GetNewChapters(ctx context.Context) ([]model.Chapter, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT c.id, c.series_id, c.chapter_num, c.title, c.url, c.release_date, c.is_new
		FROM chapters c WHERE c.is_new = true
		ORDER BY c.release_date DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get new chapters: %w", err)
	}
	defer rows.Close()

	var chapters []model.Chapter
	for rows.Next() {
		var ch model.Chapter
		if err := rows.Scan(&ch.ID, &ch.SeriesID, &ch.Number, &ch.Title,
			&ch.URL, &ch.ReleaseDate, &ch.IsNew); err != nil {
			return nil, fmt.Errorf("scan chapter: %w", err)
		}
		chapters = append(chapters, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate new chapters: %w", err)
	}
	return chapters, nil
}

func (d *DB) MarkChapterNotified(ctx context.Context, chapterID string) error {
	_, err := d.pool.Exec(ctx, `UPDATE chapters SET is_new = false WHERE id = $1`, chapterID)
	if err != nil {
		return fmt.Errorf("mark chapter notified: %w", err)
	}
	return nil
}

func (d *DB) InsertNotificationLog(ctx context.Context, chapterID, status, channel, errorMsg string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO notification_logs (chapter_id, status, channel, error_message)
		VALUES ($1, $2, $3, $4)
	`, chapterID, status, channel, errorMsg)
	if err != nil {
		return fmt.Errorf("insert notification log: %w", err)
	}
	return nil
}

func (d *DB) InsertScrapedReject(ctx context.Context, source, entityType string, payloadJSON, reasonsJSON []byte) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO scraped_rejects (source, entity_type, payload, reasons)
		VALUES ($1, $2, $3, $4)
	`, source, entityType, payloadJSON, reasonsJSON)
	if err != nil {
		return fmt.Errorf("insert scraped reject: %w", err)
	}
	return nil
}

func (d *DB) UpsertCredential(ctx context.Context, source string, payloadJSON []byte, encryptionKey string) error {
	if encryptionKey == "" {
		return fmt.Errorf("encryption key cannot be empty")
	}
	_, err := d.pool.Exec(ctx, `
		INSERT INTO source_credentials (source, encrypted_payload)
		VALUES ($1, pgp_sym_encrypt($2, $3))
		ON CONFLICT (source) DO UPDATE SET
			encrypted_payload = EXCLUDED.encrypted_payload,
			updated_at = NOW()
	`, source, string(payloadJSON), encryptionKey)

	if err != nil {
		return fmt.Errorf("upsert credential: %w", err)
	}
	return nil
}

func (d *DB) GetCredential(ctx context.Context, source string, encryptionKey string) ([]byte, error) {
	if encryptionKey == "" {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}
	var decrypted string
	err := d.pool.QueryRow(ctx, `
		SELECT pgp_sym_decrypt(encrypted_payload, $2)
		FROM source_credentials
		WHERE source = $1
	`, source, encryptionKey).Scan(&decrypted)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get credential: %w", err)
	}
	return []byte(decrypted), nil
}
