package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddFolderTable, downAddFolderTable)
}

func upAddFolderTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
create table if not exists folder(
	id varchar not null
		primary key,
	library_id integer not null
	    		references library (id)
	    		 	on delete cascade,
	path varchar default '' not null,
	name varchar default '' not null,
	updated_at timestamp default current_timestamp not null,
	created_at timestamp default current_timestamp not null,
	parent_id varchar default '' not null
);

alter table media_file 
    add column folder_id varchar default "" not null;
alter table media_file 
    add column pid varchar default id not null;
alter table media_file 
    add column album_pid varchar default album_id not null;
alter table media_file
	add column available boolean default true not null;

create index if not exists media_file_folder_id_ix
 	on media_file (folder_id);
create unique index if not exists media_file_pid_ix
	on media_file (pid);
create index if not exists media_file_album_pid_ix
	on media_file (album_pid);

alter table album
	add column folder_id varchar default "" not null;
alter table album
	add column pid varchar default id not null;

create index if not exists album_folder_id_ix
	on album (folder_id);
create unique index if not exists album_pid_ix
	on album (pid);

-- FIXME Needs to process current media_file.paths, creating folders as needed

create table if not exists tag(
  	id varchar not null primary key,
  	name varchar default '' not null,
  	value varchar default '' not null,
  	constraint tags_name_value_ux
		unique (name, value)
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

func downAddFolderTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
