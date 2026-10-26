// Package testkit provides the disposable PostgreSQL harness used by
// integration tests. It never points at a database unless TEST_DATABASE_URL is
// explicitly set and the database name looks like a test database.
package testkit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Prepare resets the public schema in the configured test database, applies all
// migrations in order, and returns a production-shaped database wrapper.
// Integration tests are skipped when TEST_DATABASE_URL is not set.
func Prepare(t *testing.T) *database.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test disabled by -short")
	}

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		if os.Getenv("TEST_REQUIRE_DB") == "1" {
			t.Fatal("TEST_DATABASE_URL is required when TEST_REQUIRE_DB=1")
		}
		t.Skip("TEST_DATABASE_URL is not set; integration test skipped")
	}
	if !strings.Contains(strings.ToLower(dsn), "test") && os.Getenv("RKU_ALLOW_DB_RESET") != "1" {
		t.Fatalf("refusing to reset a database whose DSN does not contain 'test' (set RKU_ALLOW_DB_RESET=1 only for a disposable DB)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer admin.Close()
	if err := admin.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
	if _, err := admin.Exec(ctx, "DROP SCHEMA IF EXISTS public CASCADE"); err != nil {
		t.Fatalf("drop test schema: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA public"); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	migrations, err := filepath.Glob(filepath.Join(migrationsDir(t), "*.up.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	sort.Strings(migrations)
	if len(migrations) == 0 {
		t.Fatal("no migration files found")
	}
	for _, path := range migrations {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if err := execSQLFile(ctx, admin, string(body)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}

	db, err := database.New(ctx, database.Config{URL: dsn})
	if err != nil {
		t.Fatalf("open production-shaped test database: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate testkit source")
	}
	// internal/testkit/postgres.go -> backend/scripts/postgres/migrations
	return filepath.Join(filepath.Dir(file), "..", "..", "scripts", "postgres", "migrations")
}

func execSQLFile(ctx context.Context, pool *pgxpool.Pool, body string) error {
	var cleaned strings.Builder
	for _, line := range strings.Split(body, "\n") {
		if index := strings.Index(line, "--"); index >= 0 {
			line = line[:index]
		}
		cleaned.WriteString(line)
		cleaned.WriteByte('\n')
	}
	for _, statement := range strings.Split(cleaned.String(), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := pool.Exec(ctx, statement); err != nil {
			return fmt.Errorf("%w\nstatement: %s", err, statement)
		}
	}
	return nil
}
