package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db/dialect"
	"github.com/navidrome/navidrome/log"
)

const (
	backupPrefix      = "navidrome_backup"
	backupRegexString = backupPrefix + "_(.+)\\.(db|backup)"
)

var backupRegex = regexp.MustCompile(backupRegexString)

const backupSuffixLayout = "2006.01.02_15.04.05"

func backupPath(t time.Time) string {
	ext := "db"
	if Dialect == "postgres" {
		ext = "backup"
	}
	return filepath.Join(
		conf.Server.Backup.Path,
		fmt.Sprintf("%s_%s.%s", backupPrefix, t.Format(backupSuffixLayout), ext),
	)
}

func Backup(ctx context.Context) (string, error) {
	destPath := backupPath(time.Now())
	log.Debug(ctx, "Creating backup", "path", destPath, "dialect", Dialect)
	err := dialect.Current.Backup(ctx, Db(), destPath)
	if err != nil {
		return "", err
	}

	return destPath, nil
}

func Restore(ctx context.Context, path string) error {
	log.Debug(ctx, "Restoring backup", "path", path, "dialect", Dialect)
	return dialect.Current.Restore(ctx, Db(), path)
}

func Prune(ctx context.Context) (int, error) {
	files, err := os.ReadDir(conf.Server.Backup.Path)
	if err != nil {
		return 0, fmt.Errorf("unable to read database backup entries: %w", err)
	}

	var backupTimes []time.Time

	for _, file := range files {
		if !file.IsDir() {
			submatch := backupRegex.FindStringSubmatch(file.Name())
			// submatch has 3 groups: [full match, timestamp, extension]
			if len(submatch) >= 2 {
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
