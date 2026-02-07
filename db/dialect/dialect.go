package dialect

import (
	"context"
	"database/sql"
)

// Dialect abstracts database-specific behavior (SQLite, PostgreSQL).
type Dialect interface {
	// Identity
	Name() string   // "sqlite3" or "postgres"
	Driver() string // Driver name for sql.Open

	// Connection
	DSN() string                                               // Data source name / connection string
	RegisterDriver() error                                     // Register custom driver if needed
	ConfigureConnection(ctx context.Context, db *sql.DB) error // Post-connection setup (e.g., PRAGMA for SQLite)

	// SQL Generation
	Placeholder(index int) string                          // ? for SQLite, $1 for PostgreSQL
	CaseInsensitiveComparison(column, value string) string // column LIKE value COLLATE NOCASE vs LOWER(column) = LOWER(value)
	RandomFunc() string                                    // random() for both
	SeededRandomFunc(seedKey, idColumn string) string      // SEEDEDRAND for SQLite, custom function for PostgreSQL

	// Schema
	IsSchemaEmpty(ctx context.Context, db *sql.DB) bool
	GooseDialect() string // Dialect name for goose (sqlite3, postgres)

	// Optimization
	Optimize(ctx context.Context, db *sql.DB) error
	PostSchemaChange(ctx context.Context, db *sql.DB) error    // Run after schema changes (migrations)

	// Backup
	Backup(ctx context.Context, db *sql.DB, destPath string) error
	Restore(ctx context.Context, db *sql.DB, sourcePath string) error
}

// Current is set during database initialization.
var Current Dialect
