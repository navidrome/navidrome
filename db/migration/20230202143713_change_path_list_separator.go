package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/pressly/goose/v3"
	"golang.org/x/exp/slices"
)

func init() {
	goose.AddMigrationContext(upChangePathListSeparator, downChangePathListSeparator)
}

func upChangePathListSeparator(_ context.Context, tx *sql.Tx) error {
	//nolint:gosec
	rows, err := tx.Query(`
	select album_id, group_concat(path, '` + consts.Zwsp + `') from media_file group by album_id
	`)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("update album set paths = ? where id = ?")
	if err != nil {
		return err
	}

	var id, filePaths string
	for rows.Next() {
		err = rows.Scan(&id, &filePaths)
		if err != nil {
			return err
		}

		paths := upChangePathListSeparatorDirs(filePaths)
		_, err = stmt.Exec(paths, id)
		if err != nil {
			log.Error("Error updating album's paths", "paths", paths, "id", id, err)
		}
	}
	return rows.Err()
}

func upChangePathListSeparatorDirs(filePaths string) string {
	allPaths := strings.Split(filePaths, consts.Zwsp)
	var dirs []string
	for _, p := range allPaths {
		dir, _ := filepath.Split(p)
		dirs = append(dirs, filepath.Clean(dir))
	}
	slices.Sort(dirs)
	dirs = slices.Compact(dirs)
	return strings.Join(dirs, consts.Zwsp)
}

func downChangePathListSeparator(_ context.Context, tx *sql.Tx) error {
	return nil
}
