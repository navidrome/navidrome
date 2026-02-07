package migrations

import (
	"context"
	"database/sql"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db/dialect"
	"github.com/navidrome/navidrome/log"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFixAlbumComments, downFixAlbumComments)
}

func upFixAlbumComments(_ context.Context, tx *sql.Tx) error {
	//nolint:gosec
	selectQuery := `SELECT album.id, group_concat(media_file.comment, '` + consts.Zwsp + `') FROM album, media_file WHERE media_file.album_id = album.id GROUP BY album.id;`
	updateQuery := "UPDATE album SET comment = ? WHERE id = ?"
	if dialect.Current != nil && dialect.Current.Name() == "postgres" {
		selectQuery = `SELECT album.id, string_agg(media_file.comment, '` + consts.Zwsp + `') FROM album, media_file WHERE media_file.album_id = album.id GROUP BY album.id;`
		updateQuery = "UPDATE album SET comment = $1 WHERE id = $2"
	}
	rows, err := tx.Query(selectQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	stmt, err := tx.Prepare(updateQuery)
	if err != nil {
		return err
	}
	var id string
	var comments sql.NullString

	for rows.Next() {
		err = rows.Scan(&id, &comments)
		if err != nil {
			return err
		}
		if !comments.Valid {
			continue
		}
		comment := getComment(comments.String, consts.Zwsp)
		_, err = stmt.Exec(comment, id)

		if err != nil {
			log.Error("Error setting album's comments", "albumId", id, err)
		}
	}
	return rows.Err()
}

func downFixAlbumComments(_ context.Context, tx *sql.Tx) error {
	return nil
}

func getComment(comments string, separator string) string {
	cs := strings.Split(comments, separator)
	if len(cs) == 0 {
		return ""
	}
	first := cs[0]
	for _, c := range cs[1:] {
		if first != c {
			return ""
		}
	}
	return first
}
