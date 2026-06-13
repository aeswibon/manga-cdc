package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Run applies pending SQL migrations using goose.
func Run(ctx context.Context, databaseURL string) error {
	if os.Getenv("SKIP_MIGRATIONS") == "true" {
		return nil
	}

	dir, err := resolveMigrationsDir()
	if err != nil {
		return err
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database for migrations: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database for migrations: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := baselineIfNeeded(ctx, databaseURL, db); err != nil {
		return err
	}

	if err := goose.UpContext(ctx, db, dir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func resolveMigrationsDir() (string, error) {
	if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
		return "", fmt.Errorf("MIGRATIONS_DIR is not a directory: %s", dir)
	}

	candidates := []string{
		"/migrations",
		"db/migrations",
		"../db/migrations",
		"../../db/migrations",
		"../../../db/migrations",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			abs, err := filepath.Abs(candidate)
			if err != nil {
				return candidate, nil
			}
			return abs, nil
		}
	}

	return "", errors.New("migrations directory not found; set MIGRATIONS_DIR")
}

func baselineIfNeeded(ctx context.Context, databaseURL string, db *sql.DB) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect for migration baseline: %w", err)
	}
	defer pool.Close()

	var schemaPresent bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.manga_series') IS NOT NULL`).Scan(&schemaPresent); err != nil {
		return fmt.Errorf("check existing schema: %w", err)
	}
	if !schemaPresent {
		return nil
	}

	current, err := goose.EnsureDBVersionContext(ctx, db)
	if err != nil {
		if !errors.Is(err, goose.ErrNoNextVersion) {
			return fmt.Errorf("read goose version: %w", err)
		}
		current = 0
	}
	if current > 0 {
		return nil
	}

	baselineVersion, err := detectBaselineVersion(ctx, pool)
	if err != nil {
		return fmt.Errorf("detect migration baseline: %w", err)
	}
	if baselineVersion == 0 {
		return nil
	}

	for version := int64(1); version <= baselineVersion; version++ {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)`,
			version,
		); err != nil {
			return fmt.Errorf("stamp migration baseline %d: %w", version, err)
		}
	}

	return nil
}

func detectBaselineVersion(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	checks := []struct {
		version int64
		query   string
	}{
		{4, `SELECT to_regclass('public.scraped_rejects') IS NOT NULL`},
		{1, `SELECT to_regclass('public.manga_series') IS NOT NULL`},
	}

	for _, check := range checks {
		var present bool
		if err := pool.QueryRow(ctx, check.query).Scan(&present); err != nil {
			return 0, err
		}
		if present {
			return check.version, nil
		}
	}

	return 0, nil
}
