package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAnnotationUserForeignKey, downAddAnnotationUserForeignKey)
}

func upAddAnnotationUserForeignKey(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, adaptSQL(`
CREATE TABLE IF NOT EXISTS annotation_tmp
(
	user_id     varchar(255)    not null
        REFERENCES "user"(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
	item_id     varchar(255)    default '' not null,
	item_type   varchar(255)    default '' not null,
	play_count  integer         default 0,
	play_date   datetime,
	rating      integer         default 0,
	starred     bool            default FALSE not null,
	starred_at  datetime,
	unique (user_id, item_id, item_type)
);

INSERT INTO annotation_tmp(
    user_id, item_id, item_type, play_count, play_date, rating, starred, starred_at
)
SELECT user_id, item_id, item_type, play_count, play_date, rating, starred, starred_at
FROM annotation
WHERE user_id IN (
    SELECT id FROM "user"
);

DROP TABLE annotation`)+dropCascadeIfPostgres()+`;
ALTER TABLE annotation_tmp RENAME TO annotation;

CREATE INDEX annotation_play_count
    on annotation (play_count);
CREATE INDEX annotation_play_date
    on annotation (play_date);
CREATE INDEX annotation_rating
    on annotation (rating);
CREATE INDEX annotation_starred
    on annotation (starred);
CREATE INDEX annotation_starred_at
    on annotation (starred_at);
`)
	return err
}

func downAddAnnotationUserForeignKey(_ context.Context, _ *sql.Tx) error {
	return nil
}
