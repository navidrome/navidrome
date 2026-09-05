-- +goose Up
-- Whole Navidrome schema for PostgreSQL, translated from the SQLite migrations (spike).

CREATE TABLE IF NOT EXISTS "album"
(
    id                       text                         not null
        primary key,
    name                     text           default ''    not null,
    embed_art_path           text           default ''    not null,
    album_artist             text           default ''    not null,
    min_year                 int                    default 0     not null,
    max_year                 integer                default 0     not null,
    compilation              boolean                   default FALSE not null,
    song_count               integer                default 0     not null,
    duration                 double precision                   default 0     not null,
    genre                    text           default ''    not null,
    created_at               timestamp,
    updated_at               timestamp,
    full_text                text           default '',
    album_artist_id          text           default '',
    size                     bigint                default 0     not null,
    description              text           default ''    not null,
    small_image_url          text           default ''    not null,
    medium_image_url         text           default ''    not null,
    large_image_url          text           default ''    not null,
    external_url             text           default ''    not null,
    external_info_updated_at timestamp,
    date                     text           default ''    not null,
    min_original_year        int                    default 0     not null,
    max_original_year        int                    default 0     not null,
    original_date            text           default ''    not null,
    release_date             text           default ''    not null,
    releases                 integer                default 0     not null,
    order_album_name         text  default ''    not null,
    order_album_artist_name  text  default ''    not null,
    sort_album_name          text  default ''    not null,
    sort_album_artist_name   text  default ''    not null,
    catalog_num              text                default ''    not null,
    comment                  text                default ''    not null,
    mbz_album_id             text                default ''    not null,
    mbz_album_artist_id      text                default ''    not null,
    mbz_album_type           text                default ''    not null,
    mbz_album_comment        text                default ''    not null,
    discs                    jsonb                  default '{}'  not null,
    library_id               integer                default 1     not null
        
, imported_at timestamp default '0001-01-01 00:00:00' not null, missing boolean default false not null, mbz_release_group_id text default '' not null, tags jsonb default '{}' not null, participants jsonb default '{}' not null, folder_ids jsonb default '[]' not null, explicit_status text default '' not null, average_rating double precision NOT NULL DEFAULT 0, search_participants TEXT NOT NULL DEFAULT '', search_normalized TEXT NOT NULL DEFAULT '', rg_album_gain double precision, rg_album_peak double precision);
CREATE TABLE IF NOT EXISTS "album_artists"
(
    album_id  text not null
        ,
    artist_id text not null
        ,
    role      text default '' not null,
    sub_role  text default '' not null,
    CONSTRAINT album_artists_uq UNIQUE (artist_id, album_id, role, sub_role)
);
CREATE TABLE IF NOT EXISTS album_tags(
    album_id text not null
        ,
    tag_id text not null
        ,
    CONSTRAINT album_tags_uq UNIQUE (album_id, tag_id)
);
CREATE TABLE IF NOT EXISTS "annotation"
(
	user_id     text    not null
        ,
	item_id     text    default '' not null,
	item_type   text    default '' not null,
	play_count  integer         default 0,
	play_date   timestamp,
	rating      integer         default 0,
	starred     boolean            default FALSE not null,
	starred_at  timestamp, rated_at timestamp,
	unique (user_id, item_id, item_type)
);
CREATE TABLE IF NOT EXISTS "artist"
(
    id                       text                      not null
        primary key,
    name                     text           default '' not null,
    full_text                text           default '',
    biography                text           default '' not null,
    small_image_url          text           default '' not null,
    medium_image_url         text           default '' not null,
    large_image_url          text           default '' not null,
    external_url             text           default '' not null,
    external_info_updated_at timestamp,
    order_artist_name        text  default '' not null,
    sort_artist_name         text  default '' not null,
    mbz_artist_id            text                default '' not null
, missing boolean default false not null, similar_artists jsonb default '[]' not null, updated_at timestamp default current_timestamp not null, created_at timestamp default current_timestamp not null, average_rating double precision NOT NULL DEFAULT 0, search_normalized TEXT NOT NULL DEFAULT '', uploaded_image text DEFAULT '');
CREATE TABLE IF NOT EXISTS artwork (
  hash TEXT PRIMARY KEY,
  mime TEXT NOT NULL,
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  blur_hash TEXT NOT NULL DEFAULT '',
  thumb_hash TEXT NOT NULL DEFAULT '',
  dominant_color TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS artwork_queue (
  item_kind TEXT NOT NULL,
  item_id TEXT NOT NULL,
  image_type TEXT NOT NULL DEFAULT 'primary',
  priority INTEGER NOT NULL DEFAULT 0,
  attempts INTEGER NOT NULL DEFAULT 0,
  retry_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  enqueued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, trace text NOT NULL DEFAULT '[]',
  PRIMARY KEY (item_kind, item_id, image_type)
);
CREATE TABLE IF NOT EXISTS bookmark
(
    user_id    text not null
        ,
    item_id    text not null,
    item_type  text not null,
    comment    text,
    position   integer,
    changed_by text,
    created_at timestamp,
    updated_at timestamp,
    CONSTRAINT bookmark_pk_uq UNIQUE (user_id, item_id, item_type)
);
CREATE TABLE IF NOT EXISTS folder(
	id text not null
		primary key,
	library_id integer not null
	    		,
	path text default '' not null,
	name text default '' not null,
	missing boolean default false not null,
	parent_id text default '' not null,
	num_audio_files integer default 0 not null,
	num_playlists integer default 0 not null,
	image_files jsonb default '[]' not null,
	images_updated_at timestamp default '0001-01-01 00:00:00' not null,
	updated_at timestamp default now() not null,
	created_at timestamp default now() not null
, hash text default '' not null);
CREATE TABLE IF NOT EXISTS item_artwork (
  item_kind TEXT NOT NULL,
  item_id TEXT NOT NULL,
  image_type TEXT NOT NULL DEFAULT 'primary',
  hash TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  source_path TEXT NOT NULL DEFAULT '',
  ref_mtime BIGINT NOT NULL DEFAULT 0,
  attempted_at TIMESTAMP,
  updated_at TIMESTAMP, trace text NOT NULL DEFAULT '[]', last_failure text NOT NULL DEFAULT '[]',
  PRIMARY KEY (item_kind, item_id, image_type)
);
CREATE TABLE IF NOT EXISTS library (
			id serial primary key,
			name text not null unique,
			path text not null unique,
			remote_path text null default '',
			last_scan_at timestamp not null default '0001-01-01 00:00:00',
			updated_at timestamp not null default current_timestamp,
			created_at timestamp not null default current_timestamp
		, last_scan_started_at timestamp default '0001-01-01 00:00:00' not null, full_scan_in_progress boolean default false not null, total_songs integer default 0 not null, total_albums integer default 0 not null, total_artists integer default 0 not null, total_folders integer default 0 not null, total_files integer default 0 not null, total_missing_files integer default 0 not null, total_size bigint default 0 not null, total_duration double precision DEFAULT 0, default_new_users boolean DEFAULT false);
CREATE TABLE IF NOT EXISTS "library_artist"
(
    library_id integer NOT NULL DEFAULT 1
        ,
    artist_id text NOT NULL
        ,
    stats jsonb DEFAULT '{}'::jsonb,
    CONSTRAINT library_artist_ux_uq UNIQUE (library_id, artist_id)
);
CREATE TABLE IF NOT EXISTS library_tag (
			tag_id text NOT NULL,
			library_id INTEGER NOT NULL,
			album_count INTEGER DEFAULT 0 NOT NULL,
			media_file_count INTEGER DEFAULT 0 NOT NULL,
			PRIMARY KEY (tag_id, library_id));
CREATE TABLE IF NOT EXISTS "media_file"
(
    id                      text                         not null
        primary key,
    path                    text           default ''    not null,
    title                   text           default ''    not null,
    album                   text           default ''    not null,
    artist                  text           default ''    not null,
    artist_id               text           default ''    not null,
    album_artist            text           default ''    not null,
    album_id                text           default ''    not null,
    has_cover_art           boolean                   default FALSE not null,
    track_number            integer                default 0     not null,
    disc_number             integer                default 0     not null,
    year                    integer                default 0     not null,
    size                    bigint                default 0     not null,
    suffix                  text           default ''    not null,
    duration                double precision                   default 0     not null,
    bit_rate                integer                default 0     not null,
    genre                   text           default ''    not null,
    compilation             boolean                   default FALSE not null,
    created_at              timestamp,
    updated_at              timestamp,
    full_text               text           default '',
    album_artist_id         text           default '',
    date                    text           default ''    not null,
    original_year           int                    default 0     not null,
    original_date           text           default ''    not null,
    release_year            int                    default 0     not null,
    release_date            text           default ''    not null,
    order_album_name        text  default ''    not null,
    order_album_artist_name text  default ''    not null,
    order_artist_name       text  default ''    not null,
    sort_album_name         text  default ''    not null,
    sort_artist_name        text  default ''    not null,
    sort_album_artist_name  text  default ''    not null,
    sort_title              text  default ''    not null,
    disc_subtitle           text                default ''    not null,
    catalog_num             text                default ''    not null,
    comment                 text                default ''    not null,
    order_title             text  default ''    not null,
    mbz_recording_id        text                default ''    not null,
    mbz_album_id            text                default ''    not null,
    mbz_artist_id           text                default ''    not null,
    mbz_album_artist_id     text                default ''    not null,
    mbz_album_type          text                default ''    not null,
    mbz_album_comment       text                default ''    not null,
    mbz_release_track_id    text                default ''    not null,
    channels                integer                default 0     not null,
    lyrics                  jsonb                  default '[]'  not null,
    sample_rate             integer                default 0     not null,
    library_id              integer                default 1     not null
        
, folder_id text default '' not null, pid text default '' not null, missing boolean default false not null, mbz_release_group_id text default '' not null, tags jsonb default '{}' not null, participants jsonb default '{}' not null, explicit_status text default '' not null, birth_time timestamp default current_timestamp not null, rg_album_gain double precision, rg_album_peak double precision, rg_track_gain double precision, rg_track_peak double precision, average_rating double precision NOT NULL DEFAULT 0, search_participants TEXT NOT NULL DEFAULT '', search_normalized TEXT NOT NULL DEFAULT '', codec text DEFAULT '' NOT NULL, probe_data TEXT DEFAULT '' NOT NULL, bpm integer, bit_depth integer);
CREATE TABLE IF NOT EXISTS media_file_artists(
    	media_file_id text not null
				,
    	artist_id text not null
				,
    	role text default '' not null,
    	sub_role text default '' not null,
    	CONSTRAINT artist_tracks_uq UNIQUE (artist_id, media_file_id, role, sub_role)
);
CREATE TABLE IF NOT EXISTS media_file_tags(
    media_file_id text not null
        ,
    tag_id text not null
        ,
    CONSTRAINT media_file_tags_uq UNIQUE (media_file_id, tag_id)
);
CREATE TABLE IF NOT EXISTS "player"
(
	id text not null
		primary key,
	name text not null,
	user_agent text,
	user_id text not null
		,
	client text not null,
	ip text,
	last_seen timestamp,
	max_bit_rate int default 0,
	transcoding_id text,
	report_real_path boolean default FALSE not null,
	scrobble_enabled boolean default true
);
CREATE TABLE IF NOT EXISTS "playlist"
(
    id           text                              not null
        primary key,
    name         text  default ''    not null,
    comment      text                default ''    not null,
    duration     double precision                        default 0     not null,
    song_count   integer                     default 0     not null,
    public       boolean                        default FALSE not null,
    created_at   timestamp,
    updated_at   timestamp,
    path         text                      default ''    not null,
    sync         boolean                        default false not null,
    size         bigint                     default 0     not null,
    rules        text,
    evaluated_at timestamp,
    owner_id     text                              not null
        , uploaded_image text DEFAULT '', external_image_url text DEFAULT '', average_rating double precision NOT NULL DEFAULT 0, imported_hash text default '' not null);
CREATE TABLE IF NOT EXISTS playlist_fields (
    field text not null, 
	playlist_id text not null
		);
CREATE TABLE IF NOT EXISTS "playlist_tracks"
(
	id integer default 0 not null,
	playlist_id text not null
		,
	media_file_id text not null
);
CREATE TABLE IF NOT EXISTS "playqueue"(
    id text not null,
    user_id text not null
        ,
    current integer not null default 0,
    changed_by text,
    items text,
    created_at timestamp,
    updated_at timestamp
, position integer);
CREATE TABLE IF NOT EXISTS plugin (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    manifest JSONB NOT NULL,
    config JSONB,
    users JSONB,
    all_users boolean NOT NULL DEFAULT false,
    libraries JSONB,
    all_libraries boolean NOT NULL DEFAULT false,
    enabled boolean NOT NULL DEFAULT false,
    last_error TEXT,
    sha256 TEXT NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL
, allow_write_access boolean NOT NULL DEFAULT false);
CREATE TABLE IF NOT EXISTS property
(
	id text not null
		primary key,
	value text default '' not null
);
CREATE TABLE IF NOT EXISTS "radio"
(
    id            text           not null
        primary key,
    name          text  not null
        unique,
    stream_url    text                not null,
    home_page_url text default ''     not null,
    created_at    timestamp,
    updated_at    timestamp
, uploaded_image text NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS "scrobble_buffer"
(
    user_id text NOT NULL
        ,
    service text NOT NULL,
    media_file_id text NOT NULL
        ,
    play_time timestamp NOT NULL,
    enqueue_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    id text NOT NULL DEFAULT '',
    CONSTRAINT scrobble_buffer_pk_uq UNIQUE (user_id, service, media_file_id, play_time)
);
CREATE TABLE IF NOT EXISTS "scrobbles"(
    id INTEGER PRIMARY KEY,
    media_file_id text NOT NULL
        ,
    user_id text NOT NULL 
        ,
    submission_time INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS "share"
(
    id              text not null
        primary key,
    expires_at      timestamp,
    last_visited_at timestamp,
    resource_ids    text      not null,
    created_at      timestamp,
    updated_at      timestamp,
    user_id         text not null
        ,
    downloadable boolean not null default false,
    description text not null default '',
    resource_type text not null default '',
    contents text not null default '',
    format text not null default '',
    max_bit_rate integer not null default 0,
    visit_count integer not null default 0
);
CREATE TABLE IF NOT EXISTS tag(
  	id text not null primary key,
  	tag_name text default '' not null,
  	tag_value text default '' not null,
  	CONSTRAINT tags_name_value_uq UNIQUE (tag_name, tag_value)
);
CREATE TABLE IF NOT EXISTS transcoding
(
	id text not null primary key,
	name text not null,
	target_format text not null,
	command text default '' not null,
	default_bit_rate int default 192,
	unique (name),
	unique (target_format)
);
CREATE TABLE IF NOT EXISTS user_account
(
    id             text                              not null
        primary key,
    user_name      text                default ''    not null
        unique,
    name           text  default ''    not null,
    email          text                default ''    not null,
    password       text                default ''    not null,
    is_admin       boolean                        default FALSE not null,
    last_login_at  timestamp,
    last_access_at timestamp,
    created_at     timestamp                                  not null,
    updated_at     timestamp                                  not null
, scrobble_filter text default '' not null, token_epoch INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS user_library (
			user_id text NOT NULL,
			library_id INTEGER NOT NULL,
			PRIMARY KEY (user_id, library_id));
CREATE TABLE IF NOT EXISTS "user_props"
(
	user_id text not null
		,
	key text not null,
	value text,
	constraint user_props_pk
		primary key (user_id, key)
);
CREATE INDEX IF NOT EXISTS album_alphabetical_by_artist
    on album (compilation, order_album_artist_name, order_album_name);
CREATE INDEX IF NOT EXISTS album_artist_album_id
    on album (album_artist_id);
CREATE INDEX IF NOT EXISTS album_artists_album_id
    ON album_artists (album_id);
CREATE INDEX IF NOT EXISTS album_artists_role
    ON album_artists (role);
CREATE INDEX IF NOT EXISTS album_created_at ON album(created_at, id);
CREATE INDEX IF NOT EXISTS album_genre
    on album (genre);
CREATE INDEX IF NOT EXISTS album_imported_at
	on album (imported_at);
CREATE INDEX IF NOT EXISTS album_max_year
    on album (max_year);
CREATE INDEX IF NOT EXISTS album_mbz_album_id
	on album (mbz_album_id);
CREATE INDEX IF NOT EXISTS album_mbz_album_type
    on album (mbz_album_type);
CREATE INDEX IF NOT EXISTS album_mbz_release_group_id
	on album (mbz_release_group_id);
CREATE INDEX IF NOT EXISTS album_min_year
    on album (min_year);
CREATE INDEX IF NOT EXISTS album_name
    on album (name);
CREATE INDEX IF NOT EXISTS album_order_album_artist_name
    on album (order_album_artist_name);
CREATE INDEX IF NOT EXISTS album_order_album_name
    on album (order_album_name);
CREATE INDEX IF NOT EXISTS album_size
    on album (size);
CREATE INDEX IF NOT EXISTS album_sort_album_artist_name
    on album (coalesce(nullif(sort_album_artist_name,''),order_album_artist_name) );
CREATE INDEX IF NOT EXISTS album_sort_name
    on album (coalesce(nullif(sort_album_name,''),order_album_name) );
CREATE INDEX IF NOT EXISTS album_tags_tag_id on album_tags (tag_id);
CREATE INDEX IF NOT EXISTS album_updated_at ON album(updated_at, id);
CREATE INDEX IF NOT EXISTS annotation_play_count
    on annotation (play_count);
CREATE INDEX IF NOT EXISTS annotation_play_date
    on annotation (play_date);
CREATE INDEX IF NOT EXISTS annotation_rating
    on annotation (rating);
CREATE INDEX IF NOT EXISTS annotation_starred
    on annotation (starred);
CREATE INDEX IF NOT EXISTS annotation_starred_at
    on annotation (starred_at);
CREATE INDEX IF NOT EXISTS artist_mbz_artist_id
	on artist (mbz_artist_id);
CREATE INDEX IF NOT EXISTS artist_name
    on artist (name);
CREATE INDEX IF NOT EXISTS artist_order_artist_name
    on artist (order_artist_name);
CREATE INDEX IF NOT EXISTS artist_sort_name
    on artist (coalesce(nullif(sort_artist_name,''),order_artist_name) );
CREATE INDEX IF NOT EXISTS artist_updated_at on artist (updated_at);
CREATE INDEX IF NOT EXISTS folder_parent_id on folder(parent_id);
CREATE INDEX IF NOT EXISTS idx_library_tag_library_id ON library_tag(library_id);
CREATE INDEX IF NOT EXISTS idx_library_tag_tag_id ON library_tag(tag_id);
CREATE INDEX IF NOT EXISTS idx_user_library_library_id ON user_library(library_id);
CREATE INDEX IF NOT EXISTS idx_user_library_user_id ON user_library(user_id);
CREATE INDEX IF NOT EXISTS ix_artwork_queue_drain ON artwork_queue(item_kind, priority DESC, enqueued_at, retry_at);
CREATE INDEX IF NOT EXISTS ix_item_artwork_hash ON item_artwork(hash);
CREATE INDEX IF NOT EXISTS media_file_album_artist_sort
	on media_file(order_album_artist_name, order_album_name, release_date, disc_number, track_number);
CREATE INDEX IF NOT EXISTS media_file_album_id
    on media_file (album_id);
CREATE INDEX IF NOT EXISTS media_file_album_sort
	on media_file(order_album_name, album_id, disc_number, track_number, order_artist_name, title);
CREATE INDEX IF NOT EXISTS media_file_artist_album_id
    on media_file (album_artist_id);
CREATE INDEX IF NOT EXISTS media_file_artist_id
    on media_file (artist_id);
CREATE INDEX IF NOT EXISTS media_file_artist_sort
	on media_file(order_artist_name, order_album_name, release_date, disc_number, track_number);
CREATE INDEX IF NOT EXISTS media_file_artists_media_file_id_role
    ON media_file_artists (media_file_id, role);
CREATE INDEX IF NOT EXISTS media_file_artists_role
	on media_file_artists (role);
CREATE INDEX IF NOT EXISTS media_file_bpm on media_file (bpm);
CREATE INDEX IF NOT EXISTS media_file_channels
    on media_file (channels);
CREATE INDEX IF NOT EXISTS media_file_codec ON media_file(codec);
CREATE INDEX IF NOT EXISTS media_file_created_at ON media_file(created_at, id);
CREATE INDEX IF NOT EXISTS media_file_duration
    on media_file (duration);
CREATE INDEX IF NOT EXISTS media_file_folder_id
 	on media_file (folder_id);
CREATE INDEX IF NOT EXISTS media_file_genre
    on media_file (genre);
CREATE INDEX IF NOT EXISTS media_file_mbz_release_track_id
	on media_file (mbz_release_track_id);
CREATE INDEX IF NOT EXISTS media_file_mbz_track_id
    on media_file (mbz_recording_id);
CREATE INDEX IF NOT EXISTS media_file_missing_library_id
    on media_file(missing, library_id);
CREATE INDEX IF NOT EXISTS media_file_missing_library_order_title
	on media_file(missing, library_id, order_title, id);
CREATE INDEX IF NOT EXISTS media_file_order_title
    on media_file (order_title);
CREATE INDEX IF NOT EXISTS media_file_path
    on media_file (path);
CREATE INDEX IF NOT EXISTS media_file_path_nocase
    on media_file (path );
CREATE INDEX IF NOT EXISTS media_file_pid
	on media_file (pid);
CREATE INDEX IF NOT EXISTS media_file_sample_rate
    on media_file (sample_rate);
CREATE INDEX IF NOT EXISTS media_file_tags_tag_id on media_file_tags (tag_id);
CREATE INDEX IF NOT EXISTS media_file_title
    on media_file (title);
CREATE INDEX IF NOT EXISTS media_file_track_number
    on media_file (disc_number, track_number);
CREATE INDEX IF NOT EXISTS media_file_updated_at ON media_file(updated_at, id);
CREATE INDEX IF NOT EXISTS media_file_year
    on media_file (year);
CREATE INDEX IF NOT EXISTS player_match
	on player (client, user_agent, user_id);
CREATE INDEX IF NOT EXISTS player_name
	on player (name);
CREATE INDEX IF NOT EXISTS playlist_created_at
    on playlist (created_at);
CREATE INDEX IF NOT EXISTS playlist_evaluated_at
    on playlist (evaluated_at);
CREATE UNIQUE INDEX IF NOT EXISTS playlist_fields_idx
	on playlist_fields (field, playlist_id);
CREATE INDEX IF NOT EXISTS playlist_name
    on playlist (name);
CREATE INDEX IF NOT EXISTS playlist_size
    on playlist (size);
CREATE UNIQUE INDEX IF NOT EXISTS playlist_tracks_pos
	on playlist_tracks (playlist_id, id);
CREATE INDEX IF NOT EXISTS playlist_updated_at
    on playlist (updated_at);
CREATE INDEX IF NOT EXISTS radio_name
    on radio(name);
CREATE UNIQUE INDEX IF NOT EXISTS scrobble_buffer_id_ix ON scrobble_buffer (id);
CREATE INDEX IF NOT EXISTS scrobbles_user_time ON scrobbles(user_id, submission_time);
CREATE INDEX IF NOT EXISTS user_username_password
    on user_account(user_name , password);

-- Seed rows the old migrations used to create.
INSERT INTO library (id, name, path) VALUES (1, 'Music Library', '');
SELECT setval(pg_get_serial_sequence('library', 'id'), 1);
INSERT INTO transcoding (id, name, target_format, default_bit_rate, command) VALUES
  ('mp3',  'mp3 audio',  'mp3',  192, 'ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:a:0 -b:a %bk -v 0 -f mp3 -'),
  ('opus', 'opus audio', 'opus', 128, 'ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:a:0 -b:a %bk -v 0 -c:a libopus -f opus -'),
  ('aac',  'aac audio',  'aac',  256, 'ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:a:0 -b:a %bk -v 0 -c:a aac -f adts -'),
  ('flac', 'flac audio', 'flac', 0,   'ffmpeg -ss %t -i %s -map 0:a:0 -map_metadata 0 -map_metadata 0:s:a:0 -v 0 -c:a flac -f flac -');

-- +goose Down
-- not supported
