//go:build integration

package migrate

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testDSN() string {
	if dsn := os.Getenv("DATABASE_URL_TEST"); dsn != "" {
		return dsn
	}
	return "postgres://mangacdc:mangacdc@localhost:5432/mangacdc?sslmode=disable"
}

func TestIntegration_RunAppliesSchema(t *testing.T) {
	dsn := testDSN()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS notification_logs CASCADE`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS chapters CASCADE`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS manga_series CASCADE`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS goose_db_version CASCADE`)
	_, _ = pool.Exec(ctx, `DROP FUNCTION IF EXISTS update_updated_at() CASCADE`)

	if err := Run(ctx, dsn); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.manga_series') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("check manga_series: %v", err)
	}
	if !exists {
		t.Fatal("expected manga_series table after migration")
	}

	if err := Run(ctx, dsn); err != nil {
		t.Fatalf("Run second time: %v", err)
	}
}

func TestIntegration_BaselineExistingSchema(t *testing.T) {
	dsn := testDSN()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS notification_logs CASCADE`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS chapters CASCADE`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS manga_series CASCADE`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS goose_db_version CASCADE`)
	_, _ = pool.Exec(ctx, `DROP FUNCTION IF EXISTS update_updated_at() CASCADE`)

	if err := Run(ctx, dsn); err != nil {
		t.Fatalf("initial Run: %v", err)
	}

	_, _ = pool.Exec(ctx, `DELETE FROM goose_db_version`)

	if err := Run(ctx, dsn); err != nil {
		t.Fatalf("baseline Run: %v", err)
	}

	var version int64
	if err := pool.QueryRow(ctx, `SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = true`).Scan(&version); err != nil {
		t.Fatalf("read goose version: %v", err)
	}
	if version < 1 {
		t.Fatalf("expected goose version >= 1, got %d", version)
	}
}
