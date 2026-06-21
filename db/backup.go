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
		conf.Server.Backup.Path.MustPath(),
		fmt.Sprintf("%s_%s.db", backupPrefix, t.Format(backupSuffixLayout)),
	)
}

// backupStepPageCount is the number of database pages copied per Step call.
// Bounded so that the read lock on the source database is released between
// chunks, allowing concurrent writes. Smaller values reduce lock-hold time on
// slow destination filesystems (NFS, CIFS, network drives) at the cost of more
// syscalls. See https://www.sqlite.org/c3ref/backup_finish.html and issue #5305.
const backupStepPageCount = 100

func backupOrRestore(ctx context.Context, isBackup bool, path string) error {
	// heavily inspired by https://codingrabbits.dev/posts/go_and_sqlite_backup_and_maybe_restore/
	existingConn, err := Db().Conn(ctx)
	if err != nil {
		return fmt.Errorf("getting existing connection: %w", err)
	}
	defer existingConn.Close()

	backupDb, err := sql.Open(Driver, path)
	if err != nil {
		return fmt.Errorf("opening backup database in '%s': %w", path, err)
	}
	defer backupDb.Close()

	backupConn, err := backupDb.Conn(ctx)
	if err != nil {
		return fmt.Errorf("getting backup connection: %w", err)
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

			// Iterate in bounded chunks rather than calling Step(-1). Step(-1)
			// holds the source's read lock for the entire transfer, which
			// starves concurrent writers and can fail outright on filesystems
			// with weak locking semantics (network drives, CIFS, FUSE mounts —
			// see issue #5305). Bounded steps release the lock between calls
			// and surface transient SQLITE_BUSY/SQLITE_LOCKED to the caller.
			for {
				if err := ctx.Err(); err != nil {
					return fmt.Errorf("backup canceled: %w", err)
				}

				done, err := backupOp.Step(backupStepPageCount)
				if err != nil {
					return fmt.Errorf("error during backup step (remaining=%d pages): %w", backupOp.Remaining(), err)
				}
				if done {
					break
				}
			}

			if err := backupOp.Finish(); err != nil {
				return fmt.Errorf("error finishing backup: %w", err)
			}

			return nil
		})
	})

	return err
}

func Backup(ctx context.Context) (string, error) {
	destPath := backupPath(time.Now())
	log.Debug(ctx, "Creating backup", "path", destPath)
	err := backupOrRestore(ctx, true, destPath)
	if err != nil {
		return "", err
	}

	return destPath, nil
}

// BackupTo is like Backup but writes to an explicit path. Used by callers that
// need to direct the backup to a specific location (e.g. a network share) and
// by tests that need to exercise failure paths on the destination.
func BackupTo(ctx context.Context, path string) error {
	log.Debug(ctx, "Creating backup", "path", path)
	return backupOrRestore(ctx, true, path)
}

func Restore(ctx context.Context, path string) error {
	log.Debug(ctx, "Restoring backup", "path", path)
	return backupOrRestore(ctx, false, path)
}

func Prune(ctx context.Context) (int, error) {
	backupDir, err := conf.Server.Backup.Path.Path()
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
