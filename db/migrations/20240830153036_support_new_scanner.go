package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing/fstest"
	"unicode/utf8"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSupportNewScanner, downSupportNewScanner)
}

func upSupportNewScanner(ctx context.Context, tx *sql.Tx) error {
	if err := upSupportNewScanner_CreateTableFolder(ctx, tx); err != nil {
		return err
	}
	if err := upSupportNewScanner_PopulateFolderTable(ctx, tx); err != nil {
		return err
	}
	if err := upSupportNewScanner_UpdateTableMediaFile(ctx, tx); err != nil {
		return err
	}
	if err := upSupportNewScanner_UpdateTableAlbum(ctx, tx); err != nil {
		return err
	}
	if err := upSupportNewScanner_UpdateTableArtist(ctx, tx); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
alter table library
	add column last_scan_started_at datetime default '0000-00-00 00:00:00' not null;

create table if not exists media_file_artists(
    	media_file_id varchar not null
				references media_file (id)
	    		 	on delete cascade,
    	artist_id varchar not null
				references artist (id)
	    		 	on delete cascade,
    	role varchar default '' not null,
    	sub_role varchar default '' not null,
    	constraint artist_tracks_ux
    	    			unique (artist_id, media_file_id, role)
);

create index if not exists media_file_artists_media_file_id_ix
    on media_file_artists (media_file_id);

create table if not exists album_artists(
    	album_id varchar not null
				references album (id)
	    		 	on delete cascade,
    	artist_id varchar not null
				references artist (id)
	    		 	on delete cascade,
    	role varchar default '' not null,
    	constraint album_artists_ux
    	    			unique (album_id, artist_id, role)
);

create index if not exists album_artists_album_id_ix
    on album_artists (album_id);

-- FIXME Add link all artists with role "album_artist"

create table if not exists tag(
  	id varchar not null primary key,
  	tag_name varchar default '' not null,
  	tag_value varchar default '' not null,
  	constraint tags_name_value_ux
		unique (tag_name, tag_value)
);

create table if not exists item_tags(
    item_id varchar not null,
    item_type varchar not null,
    tag_name varchar not null,
    tag_id varchar not null,
  	constraint item_tags_ux
    	unique (item_id, item_type, tag_id)
);

create index if not exists item_tag_name_ix on item_tags(item_id, tag_name)
`)
	return err
}

func upSupportNewScanner_CreateTableFolder(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
create table if not exists folder(
	id varchar not null
		primary key,
	library_id integer not null
	    		references library (id)
	    		 	on delete cascade,
	path varchar default '' not null,
	name varchar default '' not null,
	missing boolean default false not null,
	updated_at datetime default current_timestamp not null,
	created_at datetime default current_timestamp not null,
	parent_id varchar default '' not null
);`)
	return err
}

// Use paths from `media_file` table to populate `folder` table. The `folder` table must contains all paths, including
// the ones that do not contain any media_file. We can get all paths from the media_file table and then walk the
// filesystem to insert all folders into the DB, including empty parent ones.
func upSupportNewScanner_PopulateFolderTable(ctx context.Context, tx *sql.Tx) error {
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
	var f *model.Folder
	fsys := fstest.MapFS{}

	for rows.Next() {
		err = rows.Scan(&path, &lib.ID, &lib.Path)
		if err != nil {
			return err
		}

		// TODO Windows
		path = filepath.Clean(path)
		path, _ = filepath.Rel("/", path)
		fsys[path] = &fstest.MapFile{Mode: fs.ModeDir}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error loading folders from media_file table: %w", err)
	}
	if len(fsys) == 0 {
		return nil
	}

	// Finally, walk the filesystem and insert all folders into the DB.
	stmt, err := tx.PrepareContext(ctx, "insert into folder (id, library_id, path, name, parent_id) values (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	root, _ := filepath.Rel("/", lib.Path)
	err = fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			path, _ = filepath.Rel(root, path)
			f = model.NewFolder(lib, path)
			_, err = stmt.ExecContext(ctx, f.ID, lib.ID, f.Path, f.Name, f.ParentID)
			if err != nil {
				log.Error("Error writing folder to DB", "path", path, err)
			}
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("error populating folder table: %w", err)
	}

	libPathLen := utf8.RuneCountInString(lib.Path)
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
update media_file set path = substr(path,%d);`, libPathLen+2))
	if err != nil {
		return fmt.Errorf("error updating media_file path: %w", err)
	}

	return nil
}

func upSupportNewScanner_UpdateTableArtist(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table artist
	drop column album_count;
alter table artist
	drop column song_count;
drop index if exists artist_size;
alter table artist
	drop column size;
`)
	if err != nil {
		return err
	}
	err = addColumn(ctx, tx, "artist", "updated_at", "datetime", "current_time", "(select min(album.updated_at) from album where album_artist_id = artist.id)")
	if err != nil {
		return err
	}
	err = addColumn(ctx, tx, "artist", "created_at", "datetime", "current_time", "(select min(album.created_at) from album where album_artist_id = artist.id)")
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
create index if not exists artist_updated_at_ix
	on artist (updated_at);
`)
	return err
}

func upSupportNewScanner_UpdateTableAlbum(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table album
	add column scanned_at datetime default '0000-00-00 00:00:00' not null;
create index if not exists album_scanned_at_ix
	on album (scanned_at);
`)
	return err
}

func upSupportNewScanner_UpdateTableMediaFile(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `	
alter table media_file 
    add column folder_id varchar default '' not null;
alter table media_file 
    add column pid varchar default '' not null;
alter table media_file
	add column missing boolean default false not null;
update media_file 
	set pid = id where pid = '';
`)
	if err != nil {
		return err
	}

	if err = addColumn(ctx, tx, "media_file", "birth_time", "datetime", "current_timestamp", "created_at"); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
create index if not exists media_file_birth_time_ix
	on media_file (birth_time);
create index if not exists media_file_folder_id_ix
 	on media_file (folder_id);
create index if not exists media_file_pid_ix
	on media_file (pid);
create index if not exists media_file_missing_ix
	on media_file (missing);
`)

	return err
}

func downSupportNewScanner(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
