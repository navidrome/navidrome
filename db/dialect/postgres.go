package dialect

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"

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

// postgresBackup uses pg_dump to create a custom-format backup of the database.
func postgresBackup(ctx context.Context, _ *sql.DB, destPath string) error {
	pgDump, err := exec.LookPath("pg_dump")
	if err != nil {
		return fmt.Errorf("pg_dump not found in PATH — install postgresql-client")
	}

	dsn := conf.Server.DbConnectionString
	cmd := exec.CommandContext(ctx, pgDump, "--format=custom", "--file="+destPath, "--dbname="+dsn)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump failed: %w: %s", err, output)
	}

	log.Debug(ctx, "PostgreSQL backup completed", "path", destPath)
	return nil
}

// postgresRestore uses pg_restore to restore a custom-format backup into the database.
func postgresRestore(ctx context.Context, _ *sql.DB, sourcePath string) error {
	pgRestore, err := exec.LookPath("pg_restore")
	if err != nil {
		return fmt.Errorf("pg_restore not found in PATH — install postgresql-client")
	}

	dsn := conf.Server.DbConnectionString
	cmd := exec.CommandContext(ctx, pgRestore, "--clean", "--if-exists", "--dbname="+dsn, sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore failed: %w: %s", err, output)
	}

	log.Debug(ctx, "PostgreSQL restore completed", "path", sourcePath)
	return nil
}
