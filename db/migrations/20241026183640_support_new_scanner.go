package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing/fstest"
	"unicode/utf8"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/run"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSupportNewScanner, downSupportNewScanner)
}

func upSupportNewScanner(ctx context.Context, tx *sql.Tx) error {
	execute := createExecuteFunc(ctx, tx)
	addColumn := createAddColumnFunc(ctx, tx)

	return run.Sequentially(
		upSupportNewScanner_CreateTableFolder(ctx, execute),
		upSupportNewScanner_PopulateTableFolder(ctx, tx),
		upSupportNewScanner_UpdateTableMediaFile(ctx, execute, addColumn),
		upSupportNewScanner_UpdateTableAlbum(ctx, execute),
		upSupportNewScanner_UpdateTableArtist(ctx, execute, addColumn),
		execute(`
alter table library
	add column last_scan_started_at datetime default '0000-00-00 00:00:00' not null;
alter table library
	add column full_scan_in_progress boolean default false not null;

create table if not exists media_file_artists(
    	media_file_id varchar not null
				references media_file (id)
	    		 	on delete cascade,
    	artist_id varchar not null
				references artist (id)
	    		 	on delete cascade,
    	role varchar default '' not null,
    	sub_role varchar default '' not null,
    	constraint artist_tracks
    	    			unique (artist_id, media_file_id, role, sub_role)
);
create index if not exists media_file_artists_media_file_id
    on media_file_artists (media_file_id);
create index if not exists media_file_artists_role
	on media_file_artists (role);

create table if not exists album_artists(
    	album_id varchar not null
				references album (id)
	    		 	on delete cascade,
    	artist_id varchar not null
				references artist (id)
	    		 	on delete cascade,
    	role varchar default '' not null,
    	sub_role varchar default '' not null,
    	constraint album_artists
    	    			unique (album_id, artist_id, role, sub_role)
);
create index if not exists album_artists_album_id
    on album_artists (album_id);
create index if not exists album_artists_role
	on album_artists (role);

create table if not exists tag(
  	id varchar not null primary key,
  	tag_name varchar default '' not null,
  	tag_value varchar default '' not null,
  	album_count integer default 0 not null,
  	media_file_count integer default 0 not null,
  	constraint tags_name_value
		unique (tag_name, tag_value)
);

-- Genres are now stored in the tag table
drop table if exists media_file_genres;
drop table if exists album_genres;
drop table if exists artist_genres;
drop table if exists genre;

-- Drop full_text indexes, as they are not being used by SQLite
drop index if exists media_file_full_text;
drop index if exists album_full_text;
drop index if exists artist_full_text;

-- Add PID config to properties
insert into property (id, value) values ('PIDTrack', 'track_legacy') on conflict do nothing;
insert into property (id, value) values ('PIDAlbum', 'album_legacy') on conflict do nothing;
`),
		func() error {
			notice(tx, "A full scan will be triggered to populate the new tables. This may take a while.")
			return forceFullRescan(tx)
		},
	)
}

func upSupportNewScanner_CreateTableFolder(_ context.Context, execute execStmtFunc) execFunc {
	return execute(`
create table if not exists folder(
	id varchar not null
		primary key,
	library_id integer not null
	    		references library (id)
	    		 	on delete cascade,
	path varchar default '' not null,
	name varchar default '' not null,
	missing boolean default false not null,
	parent_id varchar default '' not null,
	num_audio_files integer default 0 not null,
	num_playlists integer default 0 not null,
	image_files jsonb default '[]' not null,
	images_updated_at datetime default '0000-00-00 00:00:00' not null,
	updated_at datetime default (datetime(current_timestamp, 'localtime')) not null,
	created_at datetime default (datetime(current_timestamp, 'localtime')) not null
);
create index folder_parent_id on folder(parent_id);
`)
}

// Use paths from `media_file` table to populate `folder` table. The `folder` table must contain all paths, including
// the ones that do not contain any media_file. We can get all paths from the media_file table to populate a
// fstest.MapFS{}, and then walk the filesystem to insert all folders into the DB, including empty parent ones.
func upSupportNewScanner_PopulateTableFolder(ctx context.Context, tx *sql.Tx) execFunc {
	return func() error {
		// First, get all folder paths from media_file table
		rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
select distinct rtrim(media_file.path, replace(media_file.path, '%s', '')), library_id, library.path
from media_file
join library on media_file.library_id = library.id`, string(os.PathSeparator)))
		if err != nil {
			return err
		}
		defer rows.Close()

		// Then create an in-memory filesystem with all paths
		var path string
		var lib model.Library
		fsys := fstest.MapFS{}

		for rows.Next() {
			err = rows.Scan(&path, &lib.ID, &lib.Path)
			if err != nil {
				return err
			}

			path = strings.TrimPrefix(path, filepath.Clean(lib.Path))
			path = strings.TrimPrefix(path, string(os.PathSeparator))
			path = filepath.Clean(path)
			fsys[path] = &fstest.MapFile{Mode: fs.ModeDir}
		}
		if err = rows.Err(); err != nil {
			return fmt.Errorf("error loading folders from media_file table: %w", err)
		}
		if len(fsys) == 0 {
			return nil
		}

		stmt, err := tx.PrepareContext(ctx,
			"insert into folder (id, library_id, path, name, parent_id, updated_at) values (?, ?, ?, ?, ?, '0000-00-00 00:00:00')",
		)
		if err != nil {
			return err
		}

		// Finally, walk the in-mem filesystem and insert all folders into the DB.
		err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// Don't abort the walk, just log the error
				log.Error("error walking folder to DB", "path", path, err)
				return nil
			}
			// Skip entries that are not directories
			if !d.IsDir() {
				return nil
			}

			// Create a folder in the DB
			f := model.NewFolder(lib, path)
			_, err = stmt.ExecContext(ctx, f.ID, lib.ID, f.Path, f.Name, f.ParentID)
			if err != nil {
				log.Error("error writing folder to DB", "path", path, err)
			}
			return err
		})
		if err != nil {
			return fmt.Errorf("error populating folder table: %w", err)
		}

		// Count the number of characters in the library path
		libPath := filepath.Clean(lib.Path)
		libPathLen := utf8.RuneCountInString(libPath)

		// In one go, update all paths in the media_file table, removing the library path prefix
		// and replacing any backslashes with slashes (the path separator used by the io/fs package)
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`
update media_file set path = replace(substr(path, %d), '\', '/');`, libPathLen+2))
		if err != nil {
			return fmt.Errorf("error updating media_file path: %w", err)
		}

		return nil
	}
}

func upSupportNewScanner_UpdateTableMediaFile(_ context.Context, execute execStmtFunc, addColumn addColumnFunc) execFunc {
	return func() error {
		return run.Sequentially(
			execute(`	
alter table media_file 
    add column folder_id varchar default '' not null;
alter table media_file 
    add column pid varchar default '' not null;
alter table media_file
	add column missing boolean default false not null;
alter table media_file
	add column mbz_release_group_id varchar default '' not null;
alter table media_file
	add column tags jsonb default '{}' not null;
alter table media_file
	add column participants jsonb default '{}' not null;
alter table media_file 
    add column bit_depth integer default 0 not null;
alter table media_file
	add column explicit_status varchar default '' not null;
`),
			addColumn("media_file", "birth_time", "datetime", "current_timestamp", "created_at"),
			execute(`	
update media_file 
	set pid = id where pid = '';
create index if not exists media_file_birth_time
	on media_file (birth_time);
create index if not exists media_file_folder_id
 	on media_file (folder_id);
create index if not exists media_file_pid
	on media_file (pid);
create index if not exists media_file_missing
	on media_file (missing);
`),
		)
	}
}

func upSupportNewScanner_UpdateTableAlbum(_ context.Context, execute execStmtFunc) execFunc {
	return execute(`
drop index if exists album_all_artist_ids;
alter table album
	drop column all_artist_ids;
drop index if exists album_artist;
drop index if exists album_artist_album;
alter table album
	drop column artist;
drop index if exists album_artist_id;
alter table album
	drop column artist_id;
alter table album
	add column imported_at datetime default '0000-00-00 00:00:00' not null;
alter table album
	add column missing boolean default false not null;
alter table album
	add column mbz_release_group_id varchar default '' not null;
alter table album
	add column tags jsonb default '{}' not null;
alter table album
	add column participants jsonb default '{}' not null;
alter table album
	drop column paths;
alter table album
	drop column image_files;
alter table album
	add column folder_ids jsonb default '[]' not null;
alter table album
	add column explicit_status varchar default '' not null; 
create index if not exists album_imported_at
	on album (imported_at);
create index if not exists album_mbz_release_group_id
	on album (mbz_release_group_id);
`)
}

func upSupportNewScanner_UpdateTableArtist(_ context.Context, execute execStmtFunc, addColumn addColumnFunc) execFunc {
	return func() error {
		return run.Sequentially(
			execute(`
alter table artist
	drop column album_count;
alter table artist
	drop column song_count;
drop index if exists artist_size;
alter table artist
	drop column size;
alter table artist
	add column missing boolean default false not null;
alter table artist
	add column stats jsonb default '{"albumartist":{}}' not null;
alter table artist
	drop column similar_artists;
alter table artist
	add column similar_artists jsonb default '[]' not null;
`),
			addColumn("artist", "updated_at", "datetime", "current_time", "(select min(album.updated_at) from album where album_artist_id = artist.id)"),
			addColumn("artist", "created_at", "datetime", "current_time", "(select min(album.created_at) from album where album_artist_id = artist.id)"),
			execute(`create index if not exists artist_updated_at on artist (updated_at);`),
			execute(`update artist set external_info_updated_at = '0000-00-00 00:00:00';`),
		)
	}
}

func downSupportNewScanner(context.Context, *sql.Tx) error {
	return nil
}
