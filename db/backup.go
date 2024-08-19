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

func backupPath(t *time.Time) string {
	return filepath.Join(
		conf.Server.Backup.Path,
		fmt.Sprintf("%s_%s.db", backupPrefix, t.Format(time.RFC3339)),
	)
}

func (d *db) backupOrRestore(ctx context.Context, isBackup bool, path string) error {
	// heavily inspired by https://codingrabbits.dev/posts/go_and_sqlite_backup_and_maybe_restore/
	backupDb, err := sql.Open(Driver+"_custom", path)
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

	if err == nil && !conf.Server.Backup.Bypass {
		files, err := os.ReadDir(conf.Server.Backup.Path)
		if err != nil {
			return fmt.Errorf("unable to read database backup entries: %w", err)
		}

		times := []time.Time{}

		for _, file := range files {
			if !file.IsDir() {
				submatch := backupRegex.FindStringSubmatch(file.Name())
				if len(submatch) == 2 {
					timestamp, err := time.Parse(time.RFC3339, submatch[1])
					if err == nil {
						times = append(times, timestamp)
					}
				}
			}
		}

		slices.SortFunc(times, func(a, b time.Time) int {
			return b.Compare(a)
		})

		toPrune := pruneBackups(ctx, times)

		if len(toPrune) > 0 {
			var errs []error

			for _, time := range toPrune {
				path := backupPath(&time)
				err = os.Remove(path)

				if err != nil {
					errs = append(errs, err)
				}
			}

			if len(errs) > 0 {
				err = errors.Join(errs...)
				log.Error(ctx, "Failed to delete one or more files", "errors", err)
			}
		}
	}

	return err
}

type backupSchedule struct {
	count  int
	format string
	text   string
}

var (
	backupRegex = regexp.MustCompile(backupRegexString)
)

func pruneBackups(ctx context.Context, times []time.Time) []time.Time {
	startingIdx := 0
	toPrune := []time.Time{}

	backupSchedules := []backupSchedule{
		{conf.Server.Backup.Hourly, "2006-01-02T15", "hourly"},
		{conf.Server.Backup.Daily, time.DateOnly, "daily"},
		{conf.Server.Backup.Weekly, "", "weekly"},
		{conf.Server.Backup.Monthly, "2006-Jan", "monthly"},
		{conf.Server.Backup.Yearly, "2006", "yearly"},
	}

	for _, mapping := range backupSchedules {
		if mapping.count == 0 {
			continue
		}

		var idx int
		var item time.Time
		count := 0
		prior := ""

		for idx, item = range times[startingIdx:] {
			var current string
			if mapping.text == "weekly" {
				year, week := item.ISOWeek()
				current = fmt.Sprintf("%d-%d", year, week)
			} else {
				current = item.Format(mapping.format)
			}

			if prior != current {
				prior = current
				count += 1

				log.Debug(ctx, "Keeping backup", "time", item, "rule", mapping.text, "count", count)

				if count == mapping.count {
					break
				}
			} else {
				log.Debug(ctx, "Pruning backup", "time", item)
				toPrune = append(toPrune, item)
			}
		}

		startingIdx += idx + 1
		if startingIdx >= len(times) {
			break
		}
	}

	if startingIdx < len(times) {
		toPrune = append(toPrune, times[startingIdx:]...)
	}

	return toPrune
}
