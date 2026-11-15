package migrations

import (
	"context"
	"database/sql"

	"github.com/navidrome/navidrome/conf"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRegroupMp4AsfAlbums, downRegroupMp4AsfAlbums)
}

// The raw ©day (MP4) and WM/Year (ASF) tags were releasedate aliases, so those files got a release date
// that MP3/FLAC files with the same DATE did not, and mixed-format albums split (#6467). With the aliases
// removed, album IDs that hash the release date change for these files. Clearing scanned_pid_album makes
// the scanner treat the library as having a changed PID: it rescans it in full and moves the annotations.
func upRegroupMp4AsfAlbums(ctx context.Context, tx *sql.Tx) error {
	// A custom releasedate mapping replaces the default aliases, so the change does not apply
	if tc, ok := conf.Server().Tags["releasedate"]; ok && (tc.Ignore || len(tc.Aliases) > 0) {
		return nil
	}
	// album_legacy also appends the release date to the album ID
	res, err := tx.ExecContext(ctx, `
update library set scanned_pid_album = ''
where (coalesce(nullif(pid_album, ''), ?1) like '%releasedate%' or lower(coalesce(nullif(pid_album, ''), ?1)) = 'album_legacy')
  and id in (
    select distinct library_id from media_file
    where suffix in ('m4a', 'm4b', 'm4p', 'm4r', 'm4v', 'mp4', 'wma', 'asf')
)`, conf.Server().PID.Album)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		notice(ctx, tx, "Libraries with M4A or WMA files will be rescanned in full, to fix albums split by file format")
	}
	return nil
}

func downRegroupMp4AsfAlbums(context.Context, *sql.Tx) error {
	return nil
}
