package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20201213124814, Down20201213124814)
}

func Up20201213124814(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table album
	add all_artist_ids varchar;

create index if not exists album_all_artist_ids
	on album (all_artist_ids);
`)
	if err != nil {
		return err
	}

	return updateAlbums20201213124814(tx)
}

func updateAlbums20201213124814(tx *sql.Tx) error {
	rows, err := tx.Query(`
select a.id, a.name, a.artist_id, a.album_artist_id, group_concat(mf.artist_id, ' ') 
       from album a left join media_file mf on a.id = mf.album_id group by a.id
   `)
	if err != nil {
		return err
	}
	defer rows.Close()

	stmt, err := tx.Prepare("update album set all_artist_ids = ? where id = ?")
	if err != nil {
		return err
	}

	var id, name, artistId, albumArtistId string
	var songArtistIds sql.NullString
	for rows.Next() {
		err = rows.Scan(&id, &name, &artistId, &albumArtistId, &songArtistIds)
		if err != nil {
			return err
		}
		all := utils.SanitizeStrings(artistId, albumArtistId, songArtistIds.String)
		_, err = stmt.Exec(all, id)
		if err != nil {
			log.Error("Error setting album's artist_ids", "album", name, "albumId", id, err)
		}
	}
	return rows.Err()
}

func Down20201213124814(_ context.Context, tx *sql.Tx) error {
	return nil
}
