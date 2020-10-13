package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201012210022, Down20201012210022)
}

func Up20201012210022(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table artist
	add size integer default 0 not null;
create index if not exists artist_size
	on artist(size);

alter table playlist
	add size integer default 0 not null;
create index if not exists playlist_size
	on playlist(size);

update playlist set size = ifnull((
    select sum(size)
    from media_file f
             left join playlist_tracks pt on f.id = pt.media_file_id
    where pt.playlist_id = playlist.id
), 0);`)

	if err != nil {
		return err
	}
	notice(tx, "A full rescan will be performed to calculate artists (discographies) and playlists sizes.")
	return forceFullRescan(tx)
}

func Down20201012210022(tx *sql.Tx) error {
	return nil
}
