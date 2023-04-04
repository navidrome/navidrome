package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddTimestampIndexesGo, downAddTimestampIndexesGo)
}

func upAddTimestampIndexesGo(tx *sql.Tx) error {
	_, err := tx.Exec(`
create index if not exists album_updated_at
	on album (updated_at);
create index if not exists album_created_at
	on album (created_at);
create index if not exists playlist_updated_at
	on playlist (updated_at);
create index if not exists playlist_created_at
	on playlist (created_at);
create index if not exists media_file_created_at
	on media_file (created_at);
create index if not exists media_file_updated_at
	on media_file (updated_at);
`)
	return err
}

func downAddTimestampIndexesGo(tx *sql.Tx) error {
	return nil
}
