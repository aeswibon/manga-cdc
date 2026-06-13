package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		INSERT INTO manga_series (source_id, title, alt_titles, author, artist, description,
			cover_url, status, source_url, latest_chapter, last_checked, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (source_id) DO UPDATE SET
			title = EXCLUDED.title,
			alt_titles = EXCLUDED.alt_titles,
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
	`, s.SourceID, s.Title, s.AltTitles, s.Author, s.Artist, s.Description,
		s.CoverURL, s.Status, s.SourceURL, s.LatestChapter, time.Now(), s.IsActive).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("upsert series: %w", err)
	}
	return id, nil
}

func (d *DB) UpdateSeries(ctx context.Context, s model.Series) error {
	_, err := d.pool.Exec(ctx, `
		UPDATE manga_series SET
			title = $1,
			alt_titles = $2,
			author = $3,
			artist = $4,
			description = $5,
			cover_url = $6,
			status = $7,
			source_url = $8,
			latest_chapter = $9,
			last_checked = NOW(),
			is_active = $10,
			updated_at = NOW()
		WHERE id = $11
	`, s.Title, s.AltTitles, s.Author, s.Artist, s.Description,
		s.CoverURL, s.Status, s.SourceURL, s.LatestChapter, s.IsActive, s.ID)

	if err != nil {
		return fmt.Errorf("update series: %w", err)
	}
	return nil
}

func (d *DB) GetSeriesBySourceID(ctx context.Context, sourceID string) (*model.Series, error) {
	var s model.Series
	err := d.pool.QueryRow(ctx, `
		SELECT id, source_id, title, COALESCE(alt_titles, '[]'::jsonb), author, artist,
			description, cover_url, status, source_url, latest_chapter, is_active
		FROM manga_series WHERE source_id = $1
	`, sourceID).Scan(
		&s.ID, &s.SourceID, &s.Title, &s.AltTitles, &s.Author, &s.Artist,
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
		SELECT id, source_id, title, COALESCE(alt_titles, '[]'::jsonb), author, artist,
			description, cover_url, status, source_url, latest_chapter, is_active
		FROM manga_series WHERE LOWER(TRIM(title)) = LOWER(TRIM($1))
		LIMIT 1
	`, title).Scan(
		&s.ID, &s.SourceID, &s.Title, &s.AltTitles, &s.Author, &s.Artist,
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

	valueStrings := make([]string, 0, len(chapters))
	params := make([]any, 0, len(chapters)*5)
	for i, ch := range chapters {
		base := i * 5
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,true)", base+1, base+2, base+3, base+4, base+5))
		params = append(params, seriesID, ch.Number, ch.Title, ch.URL, ch.ReleaseDate)
	}

	query := fmt.Sprintf(`
		INSERT INTO chapters (series_id,chapter_num,title,url,release_date,is_new)
		VALUES %s
		ON CONFLICT (series_id,chapter_num) DO NOTHING
		RETURNING id, chapter_num
	`, strings.Join(valueStrings, ","))

	rows, err := d.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("bulk insert chapters: %w", err)
	}
	defer rows.Close()

	byNum := make(map[float64]model.Chapter, len(chapters))
	for _, ch := range chapters {
		byNum[ch.Number] = ch
	}

	var newChapters []model.Chapter
	for rows.Next() {
		var id string
		var chapterNum float64
		if err := rows.Scan(&id, &chapterNum); err != nil {
			return nil, fmt.Errorf("scan chapter: %w", err)
		}
		if ref, ok := byNum[chapterNum]; ok {
			ref.ID = id
			ref.SeriesID = seriesID
			ref.IsNew = true
			newChapters = append(newChapters, ref)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chapters: %w", err)
	}

	return newChapters, nil
}

func (d *DB) GetActiveSeries(ctx context.Context) ([]model.Series, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, source_id, title, COALESCE(alt_titles, '[]'::jsonb), author, artist,
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
			&s.ID, &s.SourceID, &s.Title, &s.AltTitles, &s.Author, &s.Artist,
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
