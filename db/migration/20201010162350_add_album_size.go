package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20201010162350, Down20201010162350)
}

func Up20201010162350(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table album
	add size integer default 0 not null;
create index if not exists album_size
	on album(size);

update album set size = ifnull((
select sum(f.size)
from media_file f
where f.album_id = album.id
), 0)
where id not null;`)

	return err
}

func Down20201010162350(_ context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
