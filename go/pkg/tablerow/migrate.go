package tablerow

import (
	"database/sql"
	"fmt"
)

// EnsureColumn adds a column to an existing table if it is not already
// present. columnDefinition is the column portion of an
// `ALTER TABLE <table> ADD COLUMN <columnDefinition>` statement, e.g.
// `sizes VARCHAR(255) NOT NULL DEFAULT ” AFTER eligibleFor`.
//
// It is a no-op when the column already exists, so it is safe to call on
// every startup — the codebase-wide complement to CREATE TABLE IF NOT
// EXISTS, which only handles initial table creation. The existence check
// uses the connection's current schema (DATABASE()).
func EnsureColumn(r *sql.DB, w Consumer, table, column, columnDefinition string) error {
	var n int
	err := r.QueryRow(
		`SELECT COUNT(*) FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
		table, column,
	).Scan(&n)
	if err != nil {
		return fmt.Errorf("tablerow.EnsureColumn check %s.%s: %w", table, column, err)
	}
	if n > 0 {
		return nil
	}
	if err := w.Consume(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, columnDefinition)); err != nil {
		return fmt.Errorf("tablerow.EnsureColumn add %s.%s: %w", table, column, err)
	}
	return nil
}

// EnsureIndex runs alterStatement (a full `ALTER TABLE <table> ADD INDEX
// <indexName> (...)` statement) when an index named indexName does not
// already exist on table. It is a no-op when the index is present, so it
// is safe to call on every startup.
func EnsureIndex(r *sql.DB, w Consumer, table, indexName, alterStatement string) error {
	var n int
	err := r.QueryRow(
		`SELECT COUNT(*) FROM information_schema.STATISTICS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?`,
		table, indexName,
	).Scan(&n)
	if err != nil {
		return fmt.Errorf("tablerow.EnsureIndex check %s.%s: %w", table, indexName, err)
	}
	if n > 0 {
		return nil
	}
	if err := w.Consume(alterStatement); err != nil {
		return fmt.Errorf("tablerow.EnsureIndex add %s.%s: %w", table, indexName, err)
	}
	return nil
}
