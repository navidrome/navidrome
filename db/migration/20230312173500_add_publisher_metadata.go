package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddPublisherMetadata, downAddPublisherMetadata)
}

func upAddPublisherMetadata(tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table media_file
    add publisher varchar(255);
alter table album
    add publisher varchar(255);

create index if not exists media_file_publisher
	on media_file (publisher);
create index if not exists album_publisher
	on album (publisher);

create table if not exists publisher
(
  id varchar not null primary key,
  name varchar not null,  
	constraint publisher_name_ux
		unique (name)
);

create table if not exists  album_publishers
(
	album_id varchar default null not null
		references album
			on delete cascade,
	publisher_id varchar default null not null
		references publisher
			on delete cascade,
	constraint album_publisher_ux
		unique (album_id, publisher_id)
);

create table if not exists  media_file_publishers
(
	media_file_id varchar default null not null
		references media_file
			on delete cascade,
	publisher_id varchar default null not null
		references publisher
			on delete cascade,
	constraint media_file_publisher_ux
		unique (media_file_id, publisher_id)
);

create table if not exists  artist_publishers
(
	artist_id varchar default null not null
		references artist
			on delete cascade,
	publisher_id varchar default null not null
		references publisher
			on delete cascade,
	constraint artist_publisher_ux
		unique (artist_id, publisher_id)
);
`)
	if err != nil {
		return err
	}
	notice(tx, "A full rescan needs to be performed to import more tags")
	return forceFullRescan(tx)
}

func downAddPublisherMetadata(tx *sql.Tx) error {
	return nil
}
