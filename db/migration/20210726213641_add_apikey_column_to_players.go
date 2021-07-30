package migrations

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(upAddApikeyColumnToPlayers, downAddApikeyColumnToPlayers)
}

func upAddApikeyColumnToPlayers(tx *sql.Tx) error {
	_, err := tx.Exec(`alter table player
    add user_id varchar;

UPDATE player
SET user_id = (select id from user where user_name = player.user_name)
WHERE true;

create table player_dg_tmp
(
    id               varchar(255) not null
        primary key,
    name             varchar      not null,
    user_agent       varchar,
    user_id          varchar      not null
        references user (id)
            on update cascade on delete cascade,
    client           varchar      not null,
    ip_address       varchar,
    last_seen        timestamp,
    max_bit_rate     int  default 0,
    transcoding_id   varchar,
    report_real_path bool default FALSE not null,
    scrobble_enabled bool default true,
    api_key          VARCHAR(255)
);

insert into player_dg_tmp(id,
                          name,
                          user_agent,
                          user_id,
                          client,
                          ip_address,
                          last_seen,
                          max_bit_rate,
                          transcoding_id,
                          report_real_path,
                          scrobble_enabled)
select id,
       name,
       user_agent,
       user_id,
       client,
       ip_address,
       last_seen,
       max_bit_rate,
       transcoding_id,
       report_real_path,
       scrobble_enabled
from player;

drop table player;

alter table player_dg_tmp
    rename to player;

create index player_match
    on player (client, user_agent, user_id);

create index player_name
    on player (name);
`)

	return err
}

func downAddApikeyColumnToPlayers(tx *sql.Tx) error {
	_, err := tx.Exec(`alter table player
    add user_name varchar;

UPDATE player
SET user_name = (select id from user where id = player.user_id)
WHERE true;

create table player_dg_tmp
(
    id               varchar(255) not null
        primary key,
    name             varchar      not null,
    user_agent       varchar,
    user_id          varchar      not null
        references user (id)
            on update cascade on delete cascade,
    client           varchar      not null,
    ip_address       varchar,
    last_seen        timestamp,
    max_bit_rate     int  default 0,
    transcoding_id   varchar,
    report_real_path bool default FALSE not null,
    scrobble_enabled bool default true,
    api_key          VARCHAR(255)
);

insert into player_dg_tmp(id,
                          name,
                          user_agent,
                          user_name,
                          client,
                          ip_address,
                          last_seen,
                          max_bit_rate,
                          transcoding_id,
                          report_real_path,
                          scrobble_enabled)
select id,
       name,
       user_agent,
       user_name,
       client,
       ip_address,
       last_seen,
       max_bit_rate,
       transcoding_id,
       report_real_path,
       scrobble_enabled
from player;

drop table player;

alter table player_dg_tmp
    rename to player;

create index player_match
    on player (client, user_agent, user_name);

create index player_name
    on player (name);
`)

	return err
}
