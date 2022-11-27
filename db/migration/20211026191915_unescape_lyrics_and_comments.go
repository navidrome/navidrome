package migrations

import (
	"database/sql"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upUnescapeLyricsAndComments, downUnescapeLyricsAndComments)
}

func upUnescapeLyricsAndComments(tx *sql.Tx) error {
	rows, err := tx.Query(`select id, comment, lyrics, title from media_file`)
	if err != nil {
		return err
	}
	defer rows.Close()

	stmt, err := tx.Prepare("update media_file set comment = ?, lyrics = ? where id = ?")
	if err != nil {
		return err
	}

	var id, title string
	var comment, lyrics sql.NullString
	for rows.Next() {
		err = rows.Scan(&id, &comment, &lyrics, &title)
		if err != nil {
			return err
		}

		newComment := utils.SanitizeText(comment.String)
		newLyrics := utils.SanitizeText(lyrics.String)
		_, err = stmt.Exec(newComment, newLyrics, id)
		if err != nil {
			log.Error("Error unescaping media_file's lyrics and comments", "title", title, "id", id, err)
		}
	}
	return rows.Err()
}

func downUnescapeLyricsAndComments(tx *sql.Tx) error {
	return nil
}
