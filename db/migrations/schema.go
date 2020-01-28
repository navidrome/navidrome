package migrations

var schema = `
create table if not exists media_file
(
    id varchar(255) not null
        primary key,
    title varchar(255) not null,
    album varchar(255) default '' not null,
    artist varchar(255) default '' not null,
    artist_id varchar(255) default '' not null,
    album_artist varchar(255) default '' not null,
    album_id varchar(255) default '' not null,
    has_cover_art bool default FALSE not null,
    track_number integer default 0 not null,
    disc_number integer default 0 not null,
    year integer default 0 not null,
    size integer default 0 not null,
    path varchar(1024) not null,
    suffix varchar(255) default '' not null,
    duration integer default 0 not null,
    bit_rate integer default 0 not null,
    genre varchar(255) default '' not null,
    compilation bool default FALSE not null,
    created_at datetime,
    updated_at datetime
);

create index if not exists media_file_title
    on media_file (title);

create index if not exists media_file_album_id
    on media_file (album_id);

create index if not exists media_file_album
    on media_file (album);

create index if not exists media_file_artist_id
    on media_file (artist_id);

create index if not exists media_file_artist
    on media_file (artist);

create index if not exists media_file_album_artist
    on media_file (album_artist);

create index if not exists media_file_genre
    on media_file (genre);

create index if not exists media_file_year
    on media_file (year);

create index if not exists media_file_compilation
    on media_file (compilation);

create index if not exists media_file_path
    on media_file (path);

create table if not exists annotation
(
    ann_id varchar(255) not null
        primary key,
    user_id varchar(255) default '' not null,
    item_id varchar(255) default '' not null,
    item_type varchar(255) default '' not null,
    play_count integer,
    play_date datetime,
    rating integer,
    starred bool default FALSE not null,
    starred_at datetime,
    unique (user_id, item_id, item_type)
);

create index if not exists annotation_play_count
    on annotation (play_count);

create index if not exists annotation_play_date
    on annotation (play_date);

create index if not exists annotation_starred
    on annotation (starred);

create table if not exists playlist
(
    id varchar(255) not null
        primary key,
    name varchar(255) not null,
    comment varchar(255) default '' not null,
    duration integer default 0 not null,
    owner varchar(255) default '' not null,
    public bool default FALSE not null,
    tracks text not null,
    unique (owner, name)
);

create index if not exists playlist_name
    on playlist (name);

create table if not exists property
(
    id varchar(255) not null
        primary key,
    value varchar(1024) default '' not null
);

create table if not exists search
(
    id varchar(255) not null
        primary key,
    "table" varchar(255) not null,
    full_text varchar(1024) not null
);

create index if not exists search_full_text
    on search (full_text);

create index if not exists search_table
    on search ("table");

create table if not exists user
(
    id varchar(255) not null
        primary key,
    user_name varchar(255) default '' not null
        unique,
    name varchar(255) default '' not null,
    email varchar(255) default '' not null
        unique,
    password varchar(255) default '' not null,
    is_admin bool default FALSE not null,
    last_login_at datetime,
    last_access_at datetime,
    created_at datetime not null,
    updated_at datetime not null
);
`
