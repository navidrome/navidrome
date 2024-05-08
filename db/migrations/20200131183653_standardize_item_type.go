package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20200131183653, Down20200131183653)
}

func Up20200131183653(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table search_dg_tmp
(
	id varchar(255) not null
		primary key,
	item_type varchar(255) default '' not null,
	full_text varchar(255) default '' not null
);

insert into search_dg_tmp(id, item_type, full_text) select id, "table", full_text from search;

drop table search;

alter table search_dg_tmp rename to search;

create index search_full_text
	on search (full_text);
create index search_table
	on search (item_type);

update annotation set item_type = 'media_file' where item_type = 'mediaFile';
`)
	return err
}

func Down20200131183653(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
create table search_dg_tmp
(
	id varchar(255) not null
		primary key,
	"table" varchar(255) default '' not null,
	full_text varchar(255) default '' not null
);

insert into search_dg_tmp(id, "table", full_text) select id, item_type, full_text from search;

drop table search;

alter table search_dg_tmp rename to search;

create index search_full_text
	on search (full_text);
create index search_table
	on search ("table");

update annotation set item_type = 'mediaFile' where item_type = 'media_file';
`)
	return err
}
