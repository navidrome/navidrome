package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

const (
	backupPrefix      = "navidrome_backup"
	backupRegexString = "^" + backupPrefix + "_(.+)\\.db$"
)

var backupRegex = regexp.MustCompile(backupRegexString)

const backupSuffixLayout = "2006.01.02_15.04.05"

func backupPath(t time.Time) string {
	return filepath.Join(
		conf.Server().Backup.Path.MustPath(),
		fmt.Sprintf("%s_%s.db", backupPrefix, t.Format(backupSuffixLayout)),
	)
}

func openBackupFile(path string) (*sql.DB, error) {
	Db() // Registers Driver, which the CLI commands haven't done at this point.
	backupDb, err := sql.Open(Driver, path)
	if err != nil {
		return nil, fmt.Errorf("opening backup database in '%s': %w", path, err)
	}
	return backupDb, nil
}

func copyDatabase(ctx context.Context, dest, source *sql.DB) error {
	// heavily inspired by https://codingrabbits.dev/posts/go_and_sqlite_backup_and_maybe_restore/
	destConn, err := dest.Conn(ctx)
	if err != nil {
		return fmt.Errorf("getting destination connection: %w", err)
	}
	defer destConn.Close()

	sourceConn, err := source.Conn(ctx)
	if err != nil {
		return fmt.Errorf("getting source connection: %w", err)
	}
	defer sourceConn.Close()

	return destConn.Raw(func(rawDest any) error {
		return sourceConn.Raw(func(rawSource any) error {
			sqliteDest, destOk := rawDest.(*sqlite3.SQLiteConn)
			if !destOk {
				return fmt.Errorf("error trying to convert destination to sqlite connection")
			}
			sqliteSource, sourceOk := rawSource.(*sqlite3.SQLiteConn)
			if !sourceOk {
				return fmt.Errorf("error trying to convert source to sqlite connection")
			}

			backupOp, err := sqliteDest.Backup("main", sqliteSource, "main")
			if err != nil {
				return fmt.Errorf("error starting sqlite backup: %w", err)
			}
			defer backupOp.Close()

			// Caution: -1 means that sqlite will hold a read lock until the operation finishes
			// This will lock out other writes that could happen at the same time
			done, err := backupOp.Step(-1)
			if err != nil {
				return fmt.Errorf("error during backup step: %w", err)
			}
			if !done {
				return fmt.Errorf("backup not done with step -1")
			}

			err = backupOp.Finish()
			if err != nil {
				return fmt.Errorf("error finishing backup: %w", err)
			}

			return nil
		})
	})
}

// checkRestoreSource runs sanity checks on a file about to be restored over the live
// database. They catch the common ways a backup goes bad, not every kind of damage.
func checkRestoreSource(ctx context.Context, backupDb *sql.DB, path string, integrityCheck bool) error {
	// SQLite opens an empty file as an empty database, and restoring it would wipe the live one.
	// user and media_file exist since the first migration, so backups of any age have them.
	var tables int
	err := backupDb.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name IN ('goose_db_version', 'user', 'media_file')").Scan(&tables)
	if err != nil {
		return fmt.Errorf("reading backup file '%s': %w", path, err)
	}
	if tables != 3 {
		return fmt.Errorf("backup file '%s' is not a Navidrome database", path)
	}

	// The table can be there with nothing in it, or with only the row goose adds when it creates it.
	var applied int
	err = backupDb.QueryRowContext(ctx,
		"SELECT count(*) FROM goose_db_version WHERE is_applied = 1 AND version_id > 0").Scan(&applied)
	if err != nil {
		return fmt.Errorf("reading backup file '%s': %w", path, err)
	}
	if applied == 0 {
		return fmt.Errorf("backup file '%s' has no applied migrations", path)
	}

	// The queries above only read the first pages. Both checks walk the whole file, and
	// integrity_check also compares indexes to their tables, which is much slower.
	check := "PRAGMA quick_check(1)"
	if integrityCheck {
		check = "PRAGMA integrity_check(1)"
	}
	var result string
	if err := backupDb.QueryRowContext(ctx, check).Scan(&result); err != nil {
		return fmt.Errorf("checking backup file '%s': %w", path, err)
	}
	if result != "ok" {
		return fmt.Errorf("backup file '%s' is damaged: %s", path, result)
	}
	return nil
}

func Backup(ctx context.Context) (string, error) {
	destPath := backupPath(time.Now())
	log.Debug(ctx, "Creating backup", "path", destPath)
	backupDb, err := openBackupFile(destPath)
	if err != nil {
		return "", err
	}
	defer backupDb.Close()

	if err := copyDatabase(ctx, backupDb, Db()); err != nil {
		return "", err
	}

	return destPath, nil
}

func Restore(ctx context.Context, path string, integrityCheck bool) error {
	log.Debug(ctx, "Restoring backup", "path", path)

	// The driver splits the DSN at '?', so such a path would open a different file.
	if strings.ContainsRune(path, '?') {
		return fmt.Errorf("backup path cannot contain '?': %s", path)
	}
	// The driver opens with SQLITE_OPEN_CREATE, so without this check a typo in the
	// path would create an empty database and "restore" it over the live one.
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("backup file not available: %w", err)
	}

	backupDb, err := openBackupFile(path)
	if err != nil {
		return err
	}
	defer backupDb.Close()

	if err := checkRestoreSource(ctx, backupDb, path, integrityCheck); err != nil {
		return err
	}
	return copyDatabase(ctx, Db(), backupDb)
}

func Prune(ctx context.Context) (int, error) {
	backupDir, err := conf.Server().Backup.Path.Path()
	if err != nil {
		return 0, fmt.Errorf("backup directory not available: %w", err)
	}
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return 0, fmt.Errorf("unable to read database backup entries: %w", err)
	}

	var backupTimes []time.Time

	for _, file := range files {
		if !file.IsDir() {
			submatch := backupRegex.FindStringSubmatch(file.Name())
			if len(submatch) == 2 {
				timestamp, err := time.Parse(backupSuffixLayout, submatch[1])
				if err == nil {
					backupTimes = append(backupTimes, timestamp)
				}
			}
		}
	}

	if len(backupTimes) <= conf.Server().Backup.Count {
		return 0, nil
	}

	slices.SortFunc(backupTimes, func(a, b time.Time) int {
		return b.Compare(a)
	})

	pruneCount := 0
	var errs []error

	for _, timeToPrune := range backupTimes[conf.Server().Backup.Count:] {
		log.Debug(ctx, "Pruning backup", "time", timeToPrune)
		path := backupPath(timeToPrune)
		err = os.Remove(path)
		if err != nil {
			errs = append(errs, err)
		} else {
			pruneCount++
		}
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
		log.Error(ctx, "Failed to delete one or more files", "errors", err)
	}

	return pruneCount, err
}
