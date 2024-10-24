-- +goose Up
--region Artist Table
create table artist_dg_tmp
(
    id                       varchar(255)                      not null
        primary key,
    name                     varchar(255)           default '' not null,
    album_count              integer                default 0  not null,
    full_text                varchar(255)           default '',
    song_count               integer                default 0  not null,
    size                     integer                default 0  not null,
    biography                varchar(255)           default '' not null,
    small_image_url          varchar(255)           default '' not null,
    medium_image_url         varchar(255)           default '' not null,
    large_image_url          varchar(255)           default '' not null,
    similar_artists          varchar(255)           default '' not null,
    external_url             varchar(255)           default '' not null,
    external_info_updated_at datetime,
    order_artist_name        varchar collate NOCASE default '' not null,
    sort_artist_name         varchar collate NOCASE default '' not null,
    mbz_artist_id            varchar                default '' not null
);

insert into artist_dg_tmp(id, name, album_count, full_text, song_count, size, biography, small_image_url,
                          medium_image_url, large_image_url, similar_artists, external_url, external_info_updated_at,
                          order_artist_name, sort_artist_name, mbz_artist_id)
select id,
       name,
       album_count,
       full_text,
       song_count,
       size,
       biography,
       small_image_url,
       medium_image_url,
       large_image_url,
       similar_artists,
       external_url,
       external_info_updated_at,
       order_artist_name,
       sort_artist_name,
       mbz_artist_id
from artist;

drop table artist;

alter table artist_dg_tmp
    rename to artist;

create index artist_full_text
    on artist (full_text);

create index artist_name
    on artist (name);

create index artist_order_artist_name
    on artist (order_artist_name);

create index artist_size
    on artist (size);

create index artist_sort_name
    on artist (coalesce(nullif(sort_artist_name,''),order_artist_name) collate NOCASE);

--endregion

--region Album Table
create table album_dg_tmp
(
    id                       varchar(255)                         not null
        primary key,
    name                     varchar(255)           default ''    not null,
    artist_id                varchar(255)           default ''    not null,
    embed_art_path           varchar(255)           default ''    not null,
    artist                   varchar(255)           default ''    not null,
    album_artist             varchar(255)           default ''    not null,
    min_year                 int                    default 0     not null,
    max_year                 integer                default 0     not null,
    compilation              bool                   default FALSE not null,
    song_count               integer                default 0     not null,
    duration                 real                   default 0     not null,
    genre                    varchar(255)           default ''    not null,
    created_at               datetime,
    updated_at               datetime,
    full_text                varchar(255)           default '',
    album_artist_id          varchar(255)           default '',
    size                     integer                default 0     not null,
    all_artist_ids           varchar,
    description              varchar(255)           default ''    not null,
    small_image_url          varchar(255)           default ''    not null,
    medium_image_url         varchar(255)           default ''    not null,
    large_image_url          varchar(255)           default ''    not null,
    external_url             varchar(255)           default ''    not null,
    external_info_updated_at datetime,
    date                     varchar(255)           default ''    not null,
    min_original_year        int                    default 0     not null,
    max_original_year        int                    default 0     not null,
    original_date            varchar(255)           default ''    not null,
    release_date             varchar(255)           default ''    not null,
    releases                 integer                default 0     not null,
    image_files              varchar                default ''    not null,
    order_album_name         varchar collate NOCASE default ''    not null,
    order_album_artist_name  varchar collate NOCASE default ''    not null,
    sort_album_name          varchar collate NOCASE default ''    not null,
    sort_album_artist_name   varchar collate NOCASE default ''    not null,
    catalog_num              varchar                default ''    not null,
    comment                  varchar                default ''    not null,
    paths                    varchar                default ''    not null,
    mbz_album_id             varchar                default ''    not null,
    mbz_album_artist_id      varchar                default ''    not null,
    mbz_album_type           varchar                default ''    not null,
    mbz_album_comment        varchar                default ''    not null,
    discs                    jsonb                  default '{}'  not null,
    library_id               integer                default 1     not null
        references library
            on delete cascade
);

insert into album_dg_tmp(id, name, artist_id, embed_art_path, artist, album_artist, min_year, max_year, compilation,
                         song_count, duration, genre, created_at, updated_at, full_text, album_artist_id, size,
                         all_artist_ids, description, small_image_url, medium_image_url, large_image_url, external_url,
                         external_info_updated_at, date, min_original_year, max_original_year, original_date,
                         release_date, releases, image_files, order_album_name, order_album_artist_name,
                         sort_album_name, sort_album_artist_name, catalog_num, comment, paths,
                         mbz_album_id, mbz_album_artist_id, mbz_album_type, mbz_album_comment, discs, library_id)
select id,
       name,
       artist_id,
       embed_art_path,
       artist,
       album_artist,
       min_year,
       max_year,
       compilation,
       song_count,
       duration,
       genre,
       created_at,
       updated_at,
       full_text,
       album_artist_id,
       size,
       all_artist_ids,
       description,
       small_image_url,
       medium_image_url,
       large_image_url,
       external_url,
       external_info_updated_at,
       date,
       min_original_year,
       max_original_year,
       original_date,
       release_date,
       releases,
       image_files,
       order_album_name,
       order_album_artist_name,
       sort_album_name,
       sort_album_artist_name,
       catalog_num,
       comment,
       paths,
       mbz_album_id,
       mbz_album_artist_id,
       mbz_album_type,
       mbz_album_comment,
       discs,
       library_id
from album;

drop table album;

alter table album_dg_tmp
    rename to album;

create index album_all_artist_ids
    on album (all_artist_ids);

create index album_alphabetical_by_artist
    on album (compilation, order_album_artist_name, order_album_name);

create index album_artist
    on album (artist);

create index album_artist_album
    on album (artist);

create index album_artist_album_id
    on album (album_artist_id);

create index album_artist_id
    on album (artist_id);

create index album_created_at
    on album (created_at);

create index album_full_text
    on album (full_text);

create index album_genre
    on album (genre);

create index album_max_year
    on album (max_year);

create index album_mbz_album_type
    on album (mbz_album_type);

create index album_min_year
    on album (min_year);

create index album_name
    on album (name);

create index album_order_album_artist_name
    on album (order_album_artist_name);

create index album_order_album_name
    on album (order_album_name);

create index album_size
    on album (size);

create index album_sort_name
    on album (coalesce(nullif(sort_album_name,''),order_album_name) collate NOCASE);

create index album_sort_album_artist_name
    on album (coalesce(nullif(sort_album_artist_name,''),order_album_artist_name) collate NOCASE);

create index album_updated_at
    on album (updated_at);
--endregion

--region Media File Table
create table media_file_dg_tmp
(
    id                      varchar(255)                         not null
        primary key,
    path                    varchar(255)           default ''    not null,
    title                   varchar(255)           default ''    not null,
    album                   varchar(255)           default ''    not null,
    artist                  varchar(255)           default ''    not null,
    artist_id               varchar(255)           default ''    not null,
    album_artist            varchar(255)           default ''    not null,
    album_id                varchar(255)           default ''    not null,
    has_cover_art           bool                   default FALSE not null,
    track_number            integer                default 0     not null,
    disc_number             integer                default 0     not null,
    year                    integer                default 0     not null,
    size                    integer                default 0     not null,
    suffix                  varchar(255)           default ''    not null,
    duration                real                   default 0     not null,
    bit_rate                integer                default 0     not null,
    genre                   varchar(255)           default ''    not null,
    compilation             bool                   default FALSE not null,
    created_at              datetime,
    updated_at              datetime,
    full_text               varchar(255)           default '',
    album_artist_id         varchar(255)           default '',
    date                    varchar(255)           default ''    not null,
    original_year           int                    default 0     not null,
    original_date           varchar(255)           default ''    not null,
    release_year            int                    default 0     not null,
    release_date            varchar(255)           default ''    not null,
    order_album_name        varchar collate NOCASE default ''    not null,
    order_album_artist_name varchar collate NOCASE default ''    not null,
    order_artist_name       varchar collate NOCASE default ''    not null,
    sort_album_name         varchar collate NOCASE default ''    not null,
    sort_artist_name        varchar collate NOCASE default ''    not null,
    sort_album_artist_name  varchar collate NOCASE default ''    not null,
    sort_title              varchar collate NOCASE default ''    not null,
    disc_subtitle           varchar                default ''    not null,
    catalog_num             varchar                default ''    not null,
    comment                 varchar                default ''    not null,
    order_title             varchar collate NOCASE default ''    not null,
    mbz_recording_id        varchar                default ''    not null,
    mbz_album_id            varchar                default ''    not null,
    mbz_artist_id           varchar                default ''    not null,
    mbz_album_artist_id     varchar                default ''    not null,
    mbz_album_type          varchar                default ''    not null,
    mbz_album_comment       varchar                default ''    not null,
    mbz_release_track_id    varchar                default ''    not null,
    bpm                     integer                default 0     not null,
    channels                integer                default 0     not null,
    rg_album_gain           real                   default 0     not null,
    rg_album_peak           real                   default 0     not null,
    rg_track_gain           real                   default 0     not null,
    rg_track_peak           real                   default 0     not null,
    lyrics                  jsonb                  default '[]'  not null,
    sample_rate             integer                default 0     not null,
    library_id              integer                default 1     not null
        references library
            on delete cascade
);

insert into media_file_dg_tmp(id, path, title, album, artist, artist_id, album_artist, album_id, has_cover_art,
                              track_number, disc_number, year, size, suffix, duration, bit_rate, genre, compilation,
                              created_at, updated_at, full_text, album_artist_id, date, original_year, original_date,
                              release_year, release_date, order_album_name, order_album_artist_name, order_artist_name,
                              sort_album_name, sort_artist_name, sort_album_artist_name, sort_title, disc_subtitle,
                              catalog_num, comment, order_title, mbz_recording_id, mbz_album_id, mbz_artist_id,
                              mbz_album_artist_id, mbz_album_type, mbz_album_comment, mbz_release_track_id, bpm,
                              channels, rg_album_gain, rg_album_peak, rg_track_gain, rg_track_peak, lyrics, sample_rate,
                              library_id)
select id,
       path,
       title,
       album,
       artist,
       artist_id,
       album_artist,
       album_id,
       has_cover_art,
       track_number,
       disc_number,
       year,
       size,
       suffix,
       duration,
       bit_rate,
       genre,
       compilation,
       created_at,
       updated_at,
       full_text,
       album_artist_id,
       date,
       original_year,
       original_date,
       release_year,
       release_date,
       order_album_name,
       order_album_artist_name,
       order_artist_name,
       sort_album_name,
       sort_artist_name,
       sort_album_artist_name,
       sort_title,
       disc_subtitle,
       catalog_num,
       comment,
       order_title,
       mbz_recording_id,
       mbz_album_id,
       mbz_artist_id,
       mbz_album_artist_id,
       mbz_album_type,
       mbz_album_comment,
       mbz_release_track_id,
       bpm,
       channels,
       rg_album_gain,
       rg_album_peak,
       rg_track_gain,
       rg_track_peak,
       lyrics,
       sample_rate,
       library_id
from media_file;

drop table media_file;

alter table media_file_dg_tmp
    rename to media_file;

create index media_file_album_artist
    on media_file (album_artist);

create index media_file_album_id
    on media_file (album_id);

create index media_file_artist
    on media_file (artist);

create index media_file_artist_album_id
    on media_file (album_artist_id);

create index media_file_artist_id
    on media_file (artist_id);

create index media_file_bpm
    on media_file (bpm);

create index media_file_channels
    on media_file (channels);

create index media_file_created_at
    on media_file (created_at);

create index media_file_duration
    on media_file (duration);

create index media_file_full_text
    on media_file (full_text);

create index media_file_genre
    on media_file (genre);

create index media_file_mbz_track_id
    on media_file (mbz_recording_id);

create index media_file_order_album_name
    on media_file (order_album_name);

create index media_file_order_artist_name
    on media_file (order_artist_name);

create index media_file_order_title
    on media_file (order_title);

create index media_file_path
    on media_file (path);

create index media_file_path_nocase
    on media_file (path collate NOCASE);

create index media_file_sample_rate
    on media_file (sample_rate);

create index media_file_sort_title
    on media_file (coalesce(nullif(sort_title,''),order_title) collate NOCASE);

create index media_file_sort_artist_name
    on media_file (coalesce(nullif(sort_artist_name,''),order_artist_name) collate NOCASE);

create index media_file_sort_album_name
    on media_file (coalesce(nullif(sort_album_name,''),order_album_name) collate NOCASE);

create index media_file_title
    on media_file (title);

create index media_file_track_number
    on media_file (disc_number, track_number);

create index media_file_updated_at
    on media_file (updated_at);

create index media_file_year
    on media_file (year);

--endregion

--region Radio Table
create table radio_dg_tmp
(
    id            varchar(255)           not null
        primary key,
    name          varchar collate NOCASE not null
        unique,
    stream_url    varchar                not null,
    home_page_url varchar default ''     not null,
    created_at    datetime,
    updated_at    datetime
);

insert into radio_dg_tmp(id, name, stream_url, home_page_url, created_at, updated_at)
select id, name, stream_url, home_page_url, created_at, updated_at
from radio;

drop table radio;

alter table radio_dg_tmp
    rename to radio;

create index radio_name
    on radio(name);
--endregion

--region users Table
create table user_dg_tmp
(
    id             varchar(255)                              not null
        primary key,
    user_name      varchar(255)                default ''    not null
        unique,
    name           varchar(255) collate NOCASE default ''    not null,
    email          varchar(255)                default ''    not null,
    password       varchar(255)                default ''    not null,
    is_admin       bool                        default FALSE not null,
    last_login_at  datetime,
    last_access_at datetime,
    created_at     datetime                                  not null,
    updated_at     datetime                                  not null
);

insert into user_dg_tmp(id, user_name, name, email, password, is_admin, last_login_at, last_access_at, created_at,
                        updated_at)
select id,
       user_name,
       name,
       email,
       password,
       is_admin,
       last_login_at,
       last_access_at,
       created_at,
       updated_at
from user;

drop table user;

alter table user_dg_tmp
    rename to user;

create index user_username_password
    on user(user_name collate NOCASE, password);
--endregion

-- +goose Down
alter table album
    add column sort_artist_name varchar default '' not null;
