package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/db/dialect"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddLibraryStats, downAddLibraryStats)
}

func upAddLibraryStats(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, adaptSQL(`
alter table library add column total_songs integer default 0 not null;
alter table library add column total_albums integer default 0 not null;
alter table library add column total_artists integer default 0 not null;
alter table library add column total_folders integer default 0 not null;
    alter table library add column total_files integer default 0 not null;
    alter table library add column total_missing_files integer default 0 not null;
    alter table library add column total_size integer default 0 not null;
`))
	if err != nil {
		return err
	}

	jsonArrayLength := "json_array_length(image_files)"
	missingFalse := "missing = 0"
	missingTrue := "missing = 1"
	if dialect.Current != nil && dialect.Current.Name() == "postgres" {
		jsonArrayLength = "jsonb_array_length(image_files::jsonb)"
		missingFalse = "missing = false"
		missingTrue = "missing = true"
	}

	_, err = tx.ExecContext(ctx, `
update library set
    total_songs = (
        select count(*) from media_file where library_id = library.id and `+missingFalse+`
    ),
    total_albums = (select count(*) from album where library_id = library.id and `+missingFalse+`),
    total_artists = (
        select count(*) from library_artist la
        join artist a on la.artist_id = a.id
        where la.library_id = library.id and a.`+missingFalse+`
    ),
    total_folders = (select count(*) from folder where library_id = library.id and `+missingFalse+` and num_audio_files > 0),
    total_files = (
        select COALESCE(sum(num_audio_files + num_playlists + `+jsonArrayLength+`),0)
        from folder where library_id = library.id and `+missingFalse+`
    ),
    total_missing_files = (
        select count(*) from media_file where library_id = library.id and `+missingTrue+`
    ),
    total_size = (select COALESCE(sum(size),0) from album where library_id = library.id and `+missingFalse+`);
`)
	return err
}

func downAddLibraryStats(ctx context.Context, tx *sql.Tx) error {
	return nil
}
