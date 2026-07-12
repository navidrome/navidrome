package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPlaylistTrackItemType, downAddPlaylistTrackItemType)
}

// item_type distinguishes what media_file_id refers to: 'song' (a
// model.MediaFile.ID, the historical/default case) or 'podcast_episode'
// (a model.PodcastEpisode.ID). Playlists otherwise only ever referenced
// MediaFile rows; this lets a playlist_tracks row point at either without
// reworking the column itself.
func upAddPlaylistTrackItemType(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`alter table playlist_tracks add column item_type varchar(255) default 'song' not null;`)
	return err
}

func downAddPlaylistTrackItemType(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`alter table playlist_tracks drop column item_type;`)
	return err
}
