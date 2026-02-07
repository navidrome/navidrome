package dialect

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"

	// PostgreSQL driver
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresDialect struct {
	seededRandomCreated bool
}

func NewPostgres() *PostgresDialect {
	return &PostgresDialect{}
}

func (d *PostgresDialect) Name() string {
	return "postgres"
}

func (d *PostgresDialect) Driver() string {
	return "pgx"
}

func (d *PostgresDialect) DSN() string {
	return conf.Server.DbConnectionString
}

func (d *PostgresDialect) RegisterDriver() error {
	// pgx driver is registered via import side effect
	return nil
}

func (d *PostgresDialect) ConfigureConnection(ctx context.Context, db *sql.DB) error {
	if !d.seededRandomCreated {
		err := d.createSeededRandomFunction(ctx, db)
		if err != nil {
			return err
		}
		d.seededRandomCreated = true
	}
	return nil
}

// createSeededRandomFunction sets up a PostgreSQL equivalent of SQLite's SEEDEDRAND.
// Uses session variables to store seeds; returns a consistent hash for the same seed+item_id within a session.
func (d *PostgresDialect) createSeededRandomFunction(ctx context.Context, db *sql.DB) error {
	createFunc := `
CREATE OR REPLACE FUNCTION seededrand(seed_key text, item_id text)
RETURNS bigint AS $$
DECLARE
    seed text;
    setting_name text;
BEGIN
    setting_name := 'navidrome.seed_' || seed_key;
    BEGIN
        seed := current_setting(setting_name, true);
    EXCEPTION WHEN OTHERS THEN
        seed := NULL;
    END;
    IF seed IS NULL OR seed = '' THEN
        seed := md5(random()::text || clock_timestamp()::text);
        PERFORM set_config(setting_name, seed, false);
    END IF;
    RETURN ('x' || substr(md5(seed || item_id), 1, 16))::bit(64)::bigint;
END;
$$ LANGUAGE plpgsql;
`
	_, err := db.ExecContext(ctx, createFunc)
	if err != nil {
		log.Error(ctx, "Error creating seededrand function", err)
		return fmt.Errorf("failed to create seededrand function: %w", err)
	}
	log.Debug(ctx, "Created seededrand function for PostgreSQL")
	return nil
}

func (d *PostgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (d *PostgresDialect) CaseInsensitiveComparison(column, value string) string {
	return fmt.Sprintf("LOWER(%s) = LOWER(%s)", column, value)
}

func (d *PostgresDialect) RandomFunc() string {
	return "random()"
}

func (d *PostgresDialect) SeededRandomFunc(seedKey, idColumn string) string {
	return fmt.Sprintf("seededrand('%s', %s)", seedKey, idColumn)
}

func (d *PostgresDialect) IsSchemaEmpty(ctx context.Context, db *sql.DB) bool {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'goose_db_version'
		)
	`).Scan(&exists)
	if err != nil {
		log.Fatal(ctx, "Database could not be opened!", err)
	}
	return !exists
}

func (d *PostgresDialect) GooseDialect() string {
	return "postgres"
}

func (d *PostgresDialect) Optimize(ctx context.Context, db *sql.DB) error {
	log.Debug(ctx, "Running VACUUM ANALYZE on PostgreSQL")
	_, err := db.ExecContext(ctx, "VACUUM ANALYZE")
	if err != nil {
		log.Error(ctx, "Error running VACUUM ANALYZE", err)
		return err
	}
	return nil
}

func (d *PostgresDialect) PostSchemaChange(ctx context.Context, db *sql.DB) error {
	log.Debug(ctx, "Running ANALYZE after schema changes")
	_, err := db.ExecContext(ctx, "ANALYZE")
	if err != nil {
		log.Error(ctx, "Error running ANALYZE", err)
		return err
	}
	return nil
}

func (d *PostgresDialect) Backup(ctx context.Context, db *sql.DB, destPath string) error {
	return postgresBackup(ctx, db, destPath)
}

func (d *PostgresDialect) Restore(ctx context.Context, db *sql.DB, sourcePath string) error {
	return postgresRestore(ctx, db, sourcePath)
}

// postgresBackup exports all tables as CSV into a single backup file.
func postgresBackup(ctx context.Context, db *sql.DB, destPath string) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	// Write header with table list
	_, err = file.WriteString(fmt.Sprintf("-- Navidrome PostgreSQL Backup\n-- Tables: %s\n\n", strings.Join(tables, ",")))
	if err != nil {
		return fmt.Errorf("failed to write backup header: %w", err)
	}

	// Export each table
	for _, table := range tables {
		log.Debug(ctx, "Backing up table", "table", table)

		colRows, err := db.QueryContext(ctx, `
			SELECT column_name FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1
			ORDER BY ordinal_position
		`, table)
		if err != nil {
			return fmt.Errorf("failed to get columns for table %s: %w", table, err)
		}

		var columns []string
		for colRows.Next() {
			var col string
			if err := colRows.Scan(&col); err != nil {
				colRows.Close()
				return fmt.Errorf("failed to scan column name: %w", err)
			}
			columns = append(columns, col)
		}
		colRows.Close()

		// Write table marker
		_, err = file.WriteString(fmt.Sprintf("\n-- TABLE: %s\n-- COLUMNS: %s\n", table, strings.Join(columns, ",")))
		if err != nil {
			return fmt.Errorf("failed to write table header: %w", err)
		}

		// Export data
		dataRows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to query table %s: %w", table, err)
		}

		writer := csv.NewWriter(file)
		colTypes, _ := dataRows.ColumnTypes()
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		for dataRows.Next() {
			if err := dataRows.Scan(valuePtrs...); err != nil {
				dataRows.Close()
				return fmt.Errorf("failed to scan row: %w", err)
			}

			record := make([]string, len(columns))
			for i, v := range values {
				if v == nil {
					record[i] = "\\N"
				} else {
					record[i] = fmt.Sprintf("%v", v)
				}
			}
			if err := writer.Write(record); err != nil {
				dataRows.Close()
				return fmt.Errorf("failed to write record: %w", err)
			}
		}
		dataRows.Close()
		writer.Flush()

		_, err = file.WriteString("-- END TABLE\n")
		if err != nil {
			return fmt.Errorf("failed to write table footer: %w", err)
		}

		_ = colTypes // Silence unused variable warning
	}

	log.Debug(ctx, "PostgreSQL backup completed", "path", destPath)
	return nil
}

func postgresRestore(ctx context.Context, db *sql.DB, sourcePath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	log.Warn(ctx, "PostgreSQL restore from backup file is not fully implemented", "path", sourcePath)
	return fmt.Errorf("PostgreSQL restore not yet implemented")
}
