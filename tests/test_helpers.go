package tests

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/db"
)

type testingT interface {
	TempDir() string
}

func TempFileName(t testingT, prefix, suffix string) string {
	return filepath.Join(t.TempDir(), prefix+uuid.NewString()+suffix)
}

func TempFile(t testingT, prefix, suffix string) (fs.File, string, error) {
	name := TempFileName(t, prefix, suffix)
	f, err := os.Create(name)
	return f, name, err
}

func ClearDB() error {
	_, err := db.Db().WriteDB().ExecContext(context.Background(), `
			PRAGMA writable_schema = 1;
			DELETE FROM sqlite_master;
			PRAGMA writable_schema = 0;
			VACUUM;
			PRAGMA integrity_check;
		`)
	return err
}
