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
	goose.AddMigrationContext(upChangeImageFilesListSeparator, downChangeImageFilesListSeparator)
}

func upChangeImageFilesListSeparator(_ context.Context, tx *sql.Tx) error {
	rows, err := tx.Query(`select id, image_files from album`)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("update album set image_files = ? where id = ?")
	if err != nil {
		return err
	}

	var id string
	var imageFiles sql.NullString
	for rows.Next() {
		err = rows.Scan(&id, &imageFiles)
		if err != nil {
			return err
		}

		files := upChangeImageFilesListSeparatorDirs(imageFiles.String)
		if files == imageFiles.String {
			continue
		}
		_, err = stmt.Exec(files, id)
		if err != nil {
			log.Error("Error updating album's image file list", "files", files, "id", id, err)
		}
	}
	return rows.Err()
}

func upChangeImageFilesListSeparatorDirs(filePaths string) string {
	allPaths := filepath.SplitList(filePaths)
	slices.Sort(allPaths)
	allPaths = slices.Compact(allPaths)
	return strings.Join(allPaths, consts.Zwsp)
}

func downChangeImageFilesListSeparator(_ context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
