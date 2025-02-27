package tests

import (
	"context"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model/id"
)

type testingT interface {
	TempDir() string
}

func TempFileName(t testingT, prefix, suffix string) string {
	return filepath.Join(t.TempDir(), prefix+id.NewRandom()+suffix)
}

func TempFile(t testingT, prefix, suffix string) (*os.File, string, error) {
	name := TempFileName(t, prefix, suffix)
	f, err := os.Create(name)
	return f, name, err
}

// ClearDB deletes all tables and data from the database
// https://stackoverflow.com/questions/525512/drop-all-tables-command
func ClearDB() error {
	_, err := db.Db().ExecContext(context.Background(), `
			PRAGMA writable_schema = 1;
			DELETE FROM sqlite_master;
			PRAGMA writable_schema = 0;
			VACUUM;
			PRAGMA integrity_check;
		`)
	return err
}
