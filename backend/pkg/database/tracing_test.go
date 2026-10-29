package database

import (
	"strings"
	"testing"
)

func TestSQLOperationSkipsSQLCComment(t *testing.T) {
	query := "-- name: GetWorkspaceForUser :one\nSELECT id FROM workspaces WHERE id = $1"

	if got := sqlOperation(query); got != "SELECT" {
		t.Fatalf("sqlOperation() = %q, want SELECT", got)
	}
}

func TestTruncateQueryRemovesSQLCCommentAndCompressesWhitespace(t *testing.T) {
	query := "-- name: ListEntries :many\nSELECT id,\n  name\nFROM entries"

	if got := truncateQuery(query); got != "SELECT id, name FROM entries" {
		t.Fatalf("truncateQuery() = %q", got)
	}
}

func TestTruncateQueryCapsLongText(t *testing.T) {
	query := "SELECT " + strings.Repeat("x", maxQueryTextLength+10)

	if got := truncateQuery(query); len(got) != maxQueryTextLength {
		t.Fatalf("truncateQuery() length = %d, want %d", len(got), maxQueryTextLength)
	}
}
