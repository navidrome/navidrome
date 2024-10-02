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
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

const (
	backupPrefix      = "navidrome_backup"
	backupRegexString = backupPrefix + "_(.+)\\.db"
)

var backupRegex = regexp.MustCompile(backupRegexString)

const backupSuffixLayout = "2006.01.02_15.04.05"

func backupPath(t time.Time) string {
	return filepath.Join(
		conf.Server.Backup.Path,
		fmt.Sprintf("%s_%s.db", backupPrefix, t.Format(backupSuffixLayout)),
	)
}

func (d *db) backupOrRestore(ctx context.Context, isBackup bool, path string) error {
	// heavily inspired by https://codingrabbits.dev/posts/go_and_sqlite_backup_and_maybe_restore/
	backupDb, err := sql.Open(Driver, path)
	if err != nil {
		return err
	}
	defer backupDb.Close()

	existingConn, err := d.writeDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer existingConn.Close()

	backupConn, err := backupDb.Conn(ctx)
	if err != nil {
		return err
	}
	defer backupConn.Close()

	err = existingConn.Raw(func(existing any) error {
		return backupConn.Raw(func(backup any) error {
			var sourceOk, destOk bool
			var sourceConn, destConn *sqlite3.SQLiteConn

			if isBackup {
				sourceConn, sourceOk = existing.(*sqlite3.SQLiteConn)
				destConn, destOk = backup.(*sqlite3.SQLiteConn)
			} else {
				sourceConn, sourceOk = backup.(*sqlite3.SQLiteConn)
				destConn, destOk = existing.(*sqlite3.SQLiteConn)
			}

			if !sourceOk {
				return fmt.Errorf("error trying to convert source to sqlite connection")
			}
			if !destOk {
				return fmt.Errorf("error trying to convert destination to sqlite connection")
			}

			backupOp, err := destConn.Backup("main", sourceConn, "main")
			if err != nil {
				return fmt.Errorf("error starting sqlite backup: %w", err)
			}
			defer backupOp.Close()

			// Caution: -1 means that sqlite will hold a read lock until the operation finishes
			// This will lock out other writes that could happen at the same time
			done, err := backupOp.Step(-1)
			if !done {
				return fmt.Errorf("backup not done with step -1")
			}
			if err != nil {
				return fmt.Errorf("error during backup step: %w", err)
			}

			err = backupOp.Finish()
			if err != nil {
				return fmt.Errorf("error finishing backup: %w", err)
			}

			return nil
		})
	})

	return err
}

func prune(ctx context.Context) (int, error) {
	files, err := os.ReadDir(conf.Server.Backup.Path)
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

	if len(backupTimes) <= conf.Server.Backup.Count {
		return 0, nil
	}

	slices.SortFunc(backupTimes, func(a, b time.Time) int {
		return b.Compare(a)
	})

	pruneCount := 0
	var errs []error

	for _, timeToPrune := range backupTimes[conf.Server.Backup.Count:] {
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
