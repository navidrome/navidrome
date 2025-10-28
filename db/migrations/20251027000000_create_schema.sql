-- +goose Up
-- PostgreSQL Schema for Navidrome
-- Converted from SQLite schema

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS citext;

-- Property table
CREATE TABLE property (
                          id VARCHAR NOT NULL PRIMARY KEY,
                          value VARCHAR DEFAULT '' NOT NULL
);

-- Transcoding table
CREATE TABLE transcoding (
                             id VARCHAR NOT NULL PRIMARY KEY,
                             name VARCHAR NOT NULL UNIQUE,
                             target_format VARCHAR NOT NULL UNIQUE,
                             command VARCHAR DEFAULT '' NOT NULL,
                             default_bit_rate SMALLINT DEFAULT 192
);

-- User table
CREATE TABLE "user" (
                        id VARCHAR NOT NULL PRIMARY KEY,
                        user_name CITEXT NOT NULL UNIQUE,  -- CITEXT for case-insensitive username
                        name VARCHAR DEFAULT '' NOT NULL,
                        email VARCHAR DEFAULT '' NOT NULL,
                        password VARCHAR DEFAULT '' NOT NULL,
                        is_admin BOOLEAN DEFAULT FALSE NOT NULL,
                        last_login_at TIMESTAMP,
                        last_access_at TIMESTAMP,
                        created_at TIMESTAMP NOT NULL,
                        updated_at TIMESTAMP NOT NULL
);

CREATE INDEX user_username_password ON "user"(user_name, password);

-- Library table
CREATE TABLE library (
                         id SERIAL PRIMARY KEY,
                         name VARCHAR NOT NULL UNIQUE,
                         path VARCHAR NOT NULL UNIQUE,
                         remote_path VARCHAR DEFAULT '',
                         last_scan_at TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:00',
                         last_scan_started_at TIMESTAMP DEFAULT '1970-01-01 00:00:00' NOT NULL,
                         full_scan_in_progress BOOLEAN DEFAULT FALSE NOT NULL,
                         total_songs BIGINT DEFAULT 0 NOT NULL,
                         total_albums BIGINT DEFAULT 0 NOT NULL,
                         total_artists BIGINT DEFAULT 0 NOT NULL,
                         total_folders BIGINT DEFAULT 0 NOT NULL,
                         total_files BIGINT DEFAULT 0 NOT NULL,
                         total_missing_files BIGINT DEFAULT 0 NOT NULL,
                         total_size BIGINT DEFAULT 0 NOT NULL,
                         total_duration REAL DEFAULT 0,
                         default_new_users BOOLEAN DEFAULT FALSE,
                         updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Artist table
CREATE TABLE artist (
                        id VARCHAR NOT NULL PRIMARY KEY,
                        name VARCHAR DEFAULT '' NOT NULL,
                        full_text VARCHAR DEFAULT '',
                        biography VARCHAR DEFAULT '' NOT NULL,  -- VARCHAR: biographies can be long
                        small_image_url VARCHAR DEFAULT '' NOT NULL,
                        medium_image_url VARCHAR DEFAULT '' NOT NULL,
                        large_image_url VARCHAR DEFAULT '' NOT NULL,
                        external_url VARCHAR DEFAULT '' NOT NULL,
                        external_info_updated_at TIMESTAMP,
                        order_artist_name VARCHAR DEFAULT '' NOT NULL,
                        sort_artist_name VARCHAR DEFAULT '' NOT NULL,
                        mbz_artist_id VARCHAR DEFAULT '' NOT NULL,
                        missing BOOLEAN DEFAULT FALSE NOT NULL,
                        similar_artists JSONB DEFAULT '[]' NOT NULL,
                        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX artist_name ON artist(name);
CREATE INDEX artist_order_artist_name ON artist(order_artist_name);
CREATE INDEX artist_sort_name ON artist(COALESCE(NULLIF(sort_artist_name, ''), order_artist_name));
CREATE INDEX artist_updated_at ON artist(updated_at);
CREATE INDEX artist_mbz_artist_id ON artist(mbz_artist_id);

-- Library-Artist association table
CREATE TABLE library_artist (
                                library_id INT NOT NULL DEFAULT 1 REFERENCES library(id) ON DELETE CASCADE,
                                artist_id VARCHAR NOT NULL REFERENCES artist(id) ON DELETE CASCADE,
                                stats VARCHAR DEFAULT '{}',
                                CONSTRAINT library_artist_ux UNIQUE (library_id, artist_id)
);

-- Album table
CREATE TABLE album (
                       id VARCHAR NOT NULL PRIMARY KEY,
                       name VARCHAR DEFAULT '' NOT NULL,
                       embed_art_path VARCHAR DEFAULT '' NOT NULL,
                       album_artist VARCHAR DEFAULT '' NOT NULL,
                       min_year SMALLINT DEFAULT 0 NOT NULL,
                       max_year SMALLINT DEFAULT 0 NOT NULL,
                       compilation BOOLEAN DEFAULT FALSE NOT NULL,
                       song_count INT DEFAULT 0 NOT NULL,
                       duration REAL DEFAULT 0 NOT NULL,
                       genre VARCHAR DEFAULT '' NOT NULL,
                       created_at TIMESTAMP,
                       updated_at TIMESTAMP,
                       full_text VARCHAR DEFAULT '',
                       album_artist_id VARCHAR DEFAULT '',
                       size BIGINT DEFAULT 0 NOT NULL,
                       description VARCHAR DEFAULT '' NOT NULL,
                       small_image_url VARCHAR DEFAULT '' NOT NULL,
                       medium_image_url VARCHAR DEFAULT '' NOT NULL,
                       large_image_url VARCHAR DEFAULT '' NOT NULL,
                       external_url VARCHAR DEFAULT '' NOT NULL,
                       external_info_updated_at TIMESTAMP,
                       date VARCHAR DEFAULT '' NOT NULL,
                       min_original_year SMALLINT DEFAULT 0 NOT NULL,
                       max_original_year SMALLINT DEFAULT 0 NOT NULL,
                       original_date VARCHAR DEFAULT '' NOT NULL,
                       release_date VARCHAR DEFAULT '' NOT NULL,
                       order_album_name VARCHAR DEFAULT '' NOT NULL,
                       order_album_artist_name VARCHAR DEFAULT '' NOT NULL,
                       sort_album_name VARCHAR DEFAULT '' NOT NULL,
                       sort_album_artist_name VARCHAR DEFAULT '' NOT NULL,
                       catalog_num VARCHAR DEFAULT '' NOT NULL,
                       comment VARCHAR DEFAULT '' NOT NULL,
                       mbz_album_id VARCHAR DEFAULT '' NOT NULL,
                       mbz_album_artist_id VARCHAR DEFAULT '' NOT NULL,
                       mbz_album_type VARCHAR DEFAULT '' NOT NULL,
                       mbz_album_comment VARCHAR DEFAULT '' NOT NULL,
                       discs JSONB DEFAULT '{}' NOT NULL,
                       library_id INT DEFAULT 1 NOT NULL REFERENCES library ON DELETE CASCADE,
                       imported_at TIMESTAMP DEFAULT '1970-01-01 00:00:00' NOT NULL,
                       missing BOOLEAN DEFAULT FALSE NOT NULL,
                       mbz_release_group_id VARCHAR DEFAULT '' NOT NULL,
                       tags JSONB DEFAULT '{}' NOT NULL,
                       participants JSONB DEFAULT '{}' NOT NULL,
                       folder_ids JSONB DEFAULT '[]' NOT NULL,
                       explicit_status VARCHAR DEFAULT '' NOT NULL
);

CREATE INDEX album_alphabetical_by_artist ON album(compilation, order_album_artist_name, order_album_name);
CREATE INDEX album_artist_album_id ON album(album_artist_id);
CREATE INDEX album_created_at ON album(created_at);
CREATE INDEX album_genre ON album(genre);
CREATE INDEX album_max_year ON album(max_year);
CREATE INDEX album_mbz_album_type ON album(mbz_album_type);
CREATE INDEX album_min_year ON album(min_year);
CREATE INDEX album_name ON album(name);
CREATE INDEX album_order_album_artist_name ON album(order_album_artist_name);
CREATE INDEX album_order_album_name ON album(order_album_name);
CREATE INDEX album_size ON album(size);
CREATE INDEX album_sort_name ON album(COALESCE(NULLIF(sort_album_name, ''), order_album_name));
CREATE INDEX album_sort_album_artist_name ON album(COALESCE(NULLIF(sort_album_artist_name, ''), order_album_artist_name));
CREATE INDEX album_updated_at ON album(updated_at);
CREATE INDEX album_imported_at ON album(imported_at);
CREATE INDEX album_mbz_album_id ON album(mbz_album_id);
CREATE INDEX album_mbz_release_group_id ON album(mbz_release_group_id);

-- Folder table
CREATE TABLE folder (
                        id VARCHAR NOT NULL PRIMARY KEY,
                        library_id INT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
                        path VARCHAR DEFAULT '' NOT NULL,
                        name VARCHAR DEFAULT '' NOT NULL,
                        missing BOOLEAN DEFAULT FALSE NOT NULL,
                        parent_id VARCHAR DEFAULT '' NOT NULL,
                        num_audio_files INT DEFAULT 0 NOT NULL,
                        num_playlists INT DEFAULT 0 NOT NULL,
                        image_files JSONB DEFAULT '[]' NOT NULL,
                        images_updated_at TIMESTAMP DEFAULT '1970-01-01 00:00:00' NOT NULL,
                        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
                        hash VARCHAR DEFAULT '' NOT NULL
);

CREATE INDEX folder_parent_id ON folder(parent_id);

-- Media File table
CREATE TABLE media_file (
                            id VARCHAR NOT NULL PRIMARY KEY,
                            path VARCHAR DEFAULT '' NOT NULL,
                            title VARCHAR DEFAULT '' NOT NULL,
                            album VARCHAR DEFAULT '' NOT NULL,
                            artist VARCHAR DEFAULT '' NOT NULL,
                            artist_id VARCHAR DEFAULT '' NOT NULL,
                            album_artist VARCHAR DEFAULT '' NOT NULL,
                            album_id VARCHAR DEFAULT '' NOT NULL,
                            has_cover_art BOOLEAN DEFAULT FALSE NOT NULL,
                            track_number SMALLINT DEFAULT 0 NOT NULL,
                            disc_number SMALLINT DEFAULT 0 NOT NULL,
                            year SMALLINT DEFAULT 0 NOT NULL,
                            size BIGINT DEFAULT 0 NOT NULL,
                            suffix VARCHAR DEFAULT '' NOT NULL,
                            duration REAL DEFAULT 0 NOT NULL,
                            bit_rate SMALLINT DEFAULT 0 NOT NULL,
                            genre VARCHAR DEFAULT '' NOT NULL,
                            compilation BOOLEAN DEFAULT FALSE NOT NULL,
                            created_at TIMESTAMP,
                            updated_at TIMESTAMP,
                            full_text VARCHAR DEFAULT '',
                            album_artist_id VARCHAR DEFAULT '',
                            date VARCHAR DEFAULT '' NOT NULL,
                            original_year SMALLINT DEFAULT 0 NOT NULL,
                            original_date VARCHAR DEFAULT '' NOT NULL,
                            release_year SMALLINT DEFAULT 0 NOT NULL,
                            release_date VARCHAR DEFAULT '' NOT NULL,
                            order_album_name VARCHAR DEFAULT '' NOT NULL,
                            order_album_artist_name VARCHAR DEFAULT '' NOT NULL,
                            order_artist_name VARCHAR DEFAULT '' NOT NULL,
                            sort_album_name VARCHAR DEFAULT '' NOT NULL,
                            sort_artist_name VARCHAR DEFAULT '' NOT NULL,
                            sort_album_artist_name VARCHAR DEFAULT '' NOT NULL,
                            sort_title VARCHAR DEFAULT '' NOT NULL,
                            disc_subtitle VARCHAR DEFAULT '' NOT NULL,
                            catalog_num VARCHAR DEFAULT '' NOT NULL,
                            comment VARCHAR DEFAULT '' NOT NULL,
                            order_title VARCHAR DEFAULT '' NOT NULL,
                            mbz_recording_id VARCHAR DEFAULT '' NOT NULL,
                            mbz_album_id VARCHAR DEFAULT '' NOT NULL,
                            mbz_artist_id VARCHAR DEFAULT '' NOT NULL,
                            mbz_album_artist_id VARCHAR DEFAULT '' NOT NULL,
                            mbz_album_type VARCHAR DEFAULT '' NOT NULL,
                            mbz_album_comment VARCHAR DEFAULT '' NOT NULL,
                            mbz_release_track_id VARCHAR DEFAULT '' NOT NULL,
                            bpm SMALLINT DEFAULT 0 NOT NULL,
                            channels SMALLINT DEFAULT 0 NOT NULL,
                            lyrics JSONB DEFAULT '[]' NOT NULL,
                            sample_rate INT DEFAULT 0 NOT NULL,
                            library_id INT DEFAULT 1 NOT NULL REFERENCES library ON DELETE CASCADE,
                            folder_id VARCHAR DEFAULT '' NOT NULL,
                            pid VARCHAR DEFAULT '' NOT NULL,
                            missing BOOLEAN DEFAULT FALSE NOT NULL,
                            mbz_release_group_id VARCHAR DEFAULT '' NOT NULL,
                            tags JSONB DEFAULT '{}' NOT NULL,
                            participants JSONB DEFAULT '{}' NOT NULL,
                            bit_depth SMALLINT DEFAULT 0 NOT NULL,
                            explicit_status VARCHAR DEFAULT '' NOT NULL,
                            birth_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
                            rg_album_gain REAL,
                            rg_album_peak REAL,
                            rg_track_gain REAL,
                            rg_track_peak REAL
);

CREATE INDEX media_file_album_artist ON media_file(album_artist);
CREATE INDEX media_file_album_id ON media_file(album_id);
CREATE INDEX media_file_artist ON media_file(artist);
CREATE INDEX media_file_artist_album_id ON media_file(album_artist_id);
CREATE INDEX media_file_artist_id ON media_file(artist_id);
CREATE INDEX media_file_bpm ON media_file(bpm);
CREATE INDEX media_file_channels ON media_file(channels);
CREATE INDEX media_file_created_at ON media_file(created_at);
CREATE INDEX media_file_duration ON media_file(duration);
CREATE INDEX media_file_genre ON media_file(genre);
CREATE INDEX media_file_mbz_track_id ON media_file(mbz_recording_id);
CREATE INDEX media_file_order_album_name ON media_file(order_album_name);
CREATE INDEX media_file_order_artist_name ON media_file(order_artist_name);
CREATE INDEX media_file_order_title ON media_file(order_title);
CREATE INDEX media_file_path ON media_file(path);
CREATE INDEX media_file_sample_rate ON media_file(sample_rate);
CREATE INDEX media_file_sort_title ON media_file(COALESCE(NULLIF(sort_title, ''), order_title));
CREATE INDEX media_file_sort_artist_name ON media_file(COALESCE(NULLIF(sort_artist_name, ''), order_artist_name));
CREATE INDEX media_file_sort_album_name ON media_file(COALESCE(NULLIF(sort_album_name, ''), order_album_name));
CREATE INDEX media_file_title ON media_file(title);
CREATE INDEX media_file_track_number ON media_file(disc_number, track_number);
CREATE INDEX media_file_updated_at ON media_file(updated_at);
CREATE INDEX media_file_year ON media_file(year);
CREATE INDEX media_file_birth_time ON media_file(birth_time);
CREATE INDEX media_file_folder_id ON media_file(folder_id);
CREATE INDEX media_file_pid ON media_file(pid);
CREATE INDEX media_file_missing ON media_file(missing);
CREATE INDEX media_file_mbz_release_track_id ON media_file(mbz_release_track_id);

-- Media File Artists (many-to-many with roles)
CREATE TABLE media_file_artists (
                                    media_file_id VARCHAR NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
                                    artist_id VARCHAR NOT NULL REFERENCES artist(id) ON DELETE CASCADE,
                                    role VARCHAR DEFAULT '' NOT NULL,
                                    sub_role VARCHAR DEFAULT '' NOT NULL,
                                    CONSTRAINT artist_tracks UNIQUE (artist_id, media_file_id, role, sub_role)
);

CREATE INDEX media_file_artists_media_file_id ON media_file_artists(media_file_id);
CREATE INDEX media_file_artists_role ON media_file_artists(role);

-- Album Artists (many-to-many with roles)
CREATE TABLE album_artists (
                               album_id VARCHAR NOT NULL REFERENCES album(id) ON DELETE CASCADE,
                               artist_id VARCHAR NOT NULL REFERENCES artist(id) ON DELETE CASCADE,
                               role VARCHAR DEFAULT '' NOT NULL,
                               sub_role VARCHAR DEFAULT '' NOT NULL,
                               CONSTRAINT album_artists_ux UNIQUE (album_id, artist_id, role, sub_role)
);

CREATE INDEX album_artists_album_id ON album_artists(album_id);
CREATE INDEX album_artists_role ON album_artists(role);

-- Playlist table
CREATE TABLE playlist (
                          id VARCHAR NOT NULL PRIMARY KEY,
                          name VARCHAR DEFAULT '' NOT NULL,
                          comment VARCHAR DEFAULT '' NOT NULL,
                          duration REAL DEFAULT 0 NOT NULL,
                          song_count INT DEFAULT 0 NOT NULL,
                          public BOOLEAN DEFAULT FALSE NOT NULL,
                          created_at TIMESTAMP,
                          updated_at TIMESTAMP,
                          path VARCHAR DEFAULT '' NOT NULL,
                          sync BOOLEAN DEFAULT FALSE NOT NULL,
                          size BIGINT DEFAULT 0 NOT NULL,
                          rules VARCHAR,
                          evaluated_at TIMESTAMP,
                          owner_id VARCHAR NOT NULL REFERENCES "user"(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX playlist_created_at ON playlist(created_at);
CREATE INDEX playlist_evaluated_at ON playlist(evaluated_at);
CREATE INDEX playlist_name ON playlist(name);
CREATE INDEX playlist_size ON playlist(size);
CREATE INDEX playlist_updated_at ON playlist(updated_at);

-- Playlist Tracks table
CREATE TABLE playlist_tracks (
                                 id INT DEFAULT 0 NOT NULL,
                                 playlist_id VARCHAR NOT NULL REFERENCES playlist(id) ON UPDATE CASCADE ON DELETE CASCADE,
                                 media_file_id VARCHAR NOT NULL
);

CREATE UNIQUE INDEX playlist_tracks_pos ON playlist_tracks(playlist_id, id);

-- Playlist Fields table (for smart playlists)
CREATE TABLE playlist_fields (
                                 field VARCHAR NOT NULL,
                                 playlist_id VARCHAR NOT NULL REFERENCES playlist(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE UNIQUE INDEX playlist_fields_idx ON playlist_fields(field, playlist_id);

-- Radio table
CREATE TABLE radio (
                       id VARCHAR NOT NULL PRIMARY KEY,
                       name VARCHAR NOT NULL UNIQUE,
                       stream_url VARCHAR NOT NULL,
                       home_page_url VARCHAR DEFAULT '' NOT NULL,
                       created_at TIMESTAMP,
                       updated_at TIMESTAMP
);

CREATE INDEX radio_name ON radio(name);

-- Player table
CREATE TABLE player (
                        id VARCHAR NOT NULL PRIMARY KEY,
                        name VARCHAR NOT NULL,
                        user_agent VARCHAR,
                        user_id VARCHAR NOT NULL REFERENCES "user"(id) ON UPDATE CASCADE ON DELETE CASCADE,
                        client VARCHAR NOT NULL,
                        ip VARCHAR,
                        last_seen TIMESTAMP,
                        max_bit_rate SMALLINT DEFAULT 0,
                        transcoding_id VARCHAR,
                        report_real_path BOOLEAN DEFAULT FALSE NOT NULL,
                        scrobble_enabled BOOLEAN DEFAULT TRUE
);

CREATE INDEX player_match ON player(client, user_agent, user_id);
CREATE INDEX player_name ON player(name);

-- Annotation table (play counts, ratings, stars)
CREATE TABLE annotation (
                            user_id VARCHAR NOT NULL REFERENCES "user"(id) ON DELETE CASCADE ON UPDATE CASCADE,
                            item_id VARCHAR DEFAULT '' NOT NULL,
                            item_type VARCHAR DEFAULT '' NOT NULL,
                            play_count BIGINT DEFAULT 0,
                            play_date TIMESTAMP,
                            rating SMALLINT DEFAULT 0,
                            starred BOOLEAN DEFAULT FALSE NOT NULL,
                            starred_at TIMESTAMP,
                            UNIQUE (user_id, item_id, item_type)
);

CREATE INDEX annotation_play_count ON annotation(play_count);
CREATE INDEX annotation_play_date ON annotation(play_date);
CREATE INDEX annotation_rating ON annotation(rating);
CREATE INDEX annotation_starred ON annotation(starred);
CREATE INDEX annotation_starred_at ON annotation(starred_at);

-- Bookmark table
CREATE TABLE bookmark (
                          user_id VARCHAR NOT NULL REFERENCES "user"(id) ON UPDATE CASCADE ON DELETE CASCADE,
                          item_id VARCHAR NOT NULL,
                          item_type VARCHAR NOT NULL,
                          comment VARCHAR,
                          position BIGINT,
                          changed_by VARCHAR,
                          created_at TIMESTAMP,
                          updated_at TIMESTAMP,
                          CONSTRAINT bookmark_pk UNIQUE (user_id, item_id, item_type)
);

-- User Properties table
CREATE TABLE user_props (
                            user_id VARCHAR NOT NULL REFERENCES "user"(id) ON UPDATE CASCADE ON DELETE CASCADE,
                            key VARCHAR NOT NULL,
                            value VARCHAR,
                            CONSTRAINT user_props_pk PRIMARY KEY (user_id, key)
);

-- Scrobble Buffer table
CREATE TABLE scrobble_buffer (
                                 id VARCHAR NOT NULL DEFAULT '' PRIMARY KEY,
                                 user_id VARCHAR NOT NULL REFERENCES "user"(id) ON UPDATE CASCADE ON DELETE CASCADE,
                                 service VARCHAR NOT NULL,
                                 media_file_id VARCHAR NOT NULL REFERENCES media_file(id) ON UPDATE CASCADE ON DELETE CASCADE,
                                 play_time TIMESTAMP NOT NULL,
                                 enqueue_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX scrobble_buffer_unique ON scrobble_buffer(user_id, service, media_file_id, play_time);

-- Share table
CREATE TABLE share (
                       id VARCHAR NOT NULL PRIMARY KEY,
                       expires_at TIMESTAMP,
                       last_visited_at TIMESTAMP,
                       resource_ids VARCHAR NOT NULL,
                       created_at TIMESTAMP,
                       updated_at TIMESTAMP,
                       user_id VARCHAR NOT NULL REFERENCES "user"(id) ON UPDATE CASCADE ON DELETE CASCADE,
                       downloadable BOOLEAN NOT NULL DEFAULT FALSE,
                       description VARCHAR NOT NULL DEFAULT '',
                       resource_type VARCHAR NOT NULL DEFAULT '',
                       contents VARCHAR NOT NULL DEFAULT '',
                       format VARCHAR NOT NULL DEFAULT '',
                       max_bit_rate SMALLINT NOT NULL DEFAULT 0,
                       visit_count BIGINT NOT NULL DEFAULT 0
);

-- Playqueue table
CREATE TABLE playqueue (
                           id VARCHAR NOT NULL PRIMARY KEY,
                           user_id VARCHAR NOT NULL REFERENCES "user"(id) ON UPDATE CASCADE ON DELETE CASCADE,
                           current BIGINT NOT NULL DEFAULT 0,
                           position REAL,
                           changed_by VARCHAR,
                           items VARCHAR,
                           created_at TIMESTAMP,
                           updated_at TIMESTAMP
);

-- Tag table
CREATE TABLE tag (
                     id VARCHAR NOT NULL PRIMARY KEY,
                     tag_name VARCHAR DEFAULT '' NOT NULL,
                     tag_value VARCHAR DEFAULT '' NOT NULL,
                     CONSTRAINT tags_name_value UNIQUE (tag_name, tag_value)
);

-- Library Tag table (tag statistics per library)
CREATE TABLE library_tag (
                             tag_id VARCHAR NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
                             library_id INT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
                             album_count INT DEFAULT 0 NOT NULL,
                             media_file_count BIGINT DEFAULT 0 NOT NULL,
                             PRIMARY KEY (tag_id, library_id)
);

CREATE INDEX idx_library_tag_tag_id ON library_tag(tag_id);
CREATE INDEX idx_library_tag_library_id ON library_tag(library_id);

-- User Library association table (many-to-many)
CREATE TABLE user_library (
                              user_id VARCHAR NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
                              library_id INT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
                              PRIMARY KEY (user_id, library_id)
);

CREATE INDEX idx_user_library_user_id ON user_library(user_id);
CREATE INDEX idx_user_library_library_id ON user_library(library_id);

-- Insert default library
INSERT INTO library (id, name, path, default_new_users)
VALUES (1, 'Music Library', './music', true);

-- +goose Down
SELECT 1;
