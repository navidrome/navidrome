package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddLibraryStats, downAddLibraryStats)
}

func upAddLibraryStats(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table library add column total_songs integer default 0 not null;
alter table library add column total_albums integer default 0 not null;
alter table library add column total_artists integer default 0 not null;
alter table library add column total_folders integer default 0 not null;
    alter table library add column total_files integer default 0 not null;
    alter table library add column total_missing_files integer default 0 not null;
    alter table library add column total_size integer default 0 not null;
update library set
    total_songs = (
        select count(*) from media_file where library_id = library.id and missing = 0
    ),
    total_albums = (select count(*) from album where library_id = library.id and missing = 0),
    total_artists = (
        select count(*) from library_artist la 
        join artist a on la.artist_id = a.id 
        where la.library_id = library.id and a.missing = 0
    ),
    total_folders = (select count(*) from folder where library_id = library.id and missing = 0 and num_audio_files > 0),
    total_files = (
        select ifnull(sum(num_audio_files + num_playlists + json_array_length(image_files)),0)
        from folder where library_id = library.id and missing = 0
    ),
    total_missing_files = (
        select count(*) from media_file where library_id = library.id and missing = 1
    ),
    total_size = (select ifnull(sum(size),0) from album where library_id = library.id and missing = 0);
`)
	return err
}

func downAddLibraryStats(ctx context.Context, tx *sql.Tx) error {
	return nil
}
