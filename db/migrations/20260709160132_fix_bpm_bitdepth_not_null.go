package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/navidrome/navidrome/log"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFixBpmBitdepthNotNull, downFixBpmBitdepthNotNull)
}

// upFixBpmBitdepthNotNull repairs databases where media_file.bpm/bit_depth are
// still declared NOT NULL even though migration 20260612222838 (which made them
// nullable) is recorded as applied. On those instances the scanner writes a NULL
// bpm for tracks without a BPM tag and the write fails with
// "NOT NULL constraint failed: media_file.bpm" (issue #5747). Because goose
// considers the earlier migration already applied, it will never re-run it, so a
// fresh migration is required. It re-inspects the column definition and only
// rebuilds a column when it is still NOT NULL, making it a no-op on healthy DBs.
func upFixBpmBitdepthNotNull(ctx context.Context, tx *sql.Tx) error {
	repaired := false

	// bpm carries the media_file_bpm index, which must be dropped before the
	// column can be dropped and recreated after.
	notNull, err := columnIsNotNull(ctx, tx, "media_file", "bpm")
	if err != nil {
		return err
	}
	if notNull {
		if _, err = tx.ExecContext(ctx, `
drop index if exists media_file_bpm;
alter table media_file add column bpm_new integer;
update media_file set bpm_new = nullif(bpm, 0);
alter table media_file drop column bpm;
alter table media_file rename column bpm_new to bpm;
create index if not exists media_file_bpm on media_file (bpm);
`); err != nil {
			return err
		}
		repaired = true
	}

	notNull, err = columnIsNotNull(ctx, tx, "media_file", "bit_depth")
	if err != nil {
		return err
	}
	if notNull {
		if _, err = tx.ExecContext(ctx, `
alter table media_file add column bit_depth_new integer;
update media_file set bit_depth_new = nullif(bit_depth, 0);
alter table media_file drop column bit_depth;
alter table media_file rename column bit_depth_new to bit_depth;
`); err != nil {
			return err
		}
		repaired = true
	}

	if repaired {
		log.Warn(ctx, "Repaired media_file.bpm/bit_depth NOT NULL columns left over from an out-of-order migration (issue #5747)")
	}
	return nil
}

func downFixBpmBitdepthNotNull(ctx context.Context, tx *sql.Tx) error {
	// No-op: re-adding the NOT NULL constraint would reintroduce the bug.
	return nil
}

// columnIsNotNull reports whether the given column of the given table is declared
// with a NOT NULL constraint, using PRAGMA table_info. The table name is a fixed
// internal constant, so it is safe to interpolate (PRAGMA cannot bind parameters).
func columnIsNotNull(ctx context.Context, tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err = rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return notnull == 1, nil
		}
	}
	return false, rows.Err()
}
