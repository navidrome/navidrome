package tests

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
	"github.com/onsi/ginkgo/v2"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// SkipOnWindows marks the current spec (or surrounding BeforeEach) as skipped
// when running on Windows. The reason is included in the Ginkgo output so the
// backlog of Windows-skipped tests stays auditable.
func SkipOnWindows(reason string) {
	if runtime.GOOS == "windows" {
		ginkgo.Skip("not supported on Windows: " + reason)
	}
}

type testingT interface {
	TempDir() string
}

func TempFileName(t testingT, prefix, suffix string) string {
	return filepath.Join(t.TempDir(), prefix+id.NewRandom()+suffix)
}

// TempFile creates an empty file in t.TempDir() and returns the closed handle.
// The handle is returned for backward compatibility, but is already closed so
// callers don't need to. On Windows, leaving the handle open would hold a file
// lock and block Ginkgo's TempDir cleanup.
func TempFile(t testingT, prefix, suffix string) (*os.File, string, error) {
	name := TempFileName(t, prefix, suffix)
	f, err := os.Create(name)
	if err != nil {
		return nil, name, err
	}
	if cerr := f.Close(); cerr != nil {
		return f, name, cerr
	}
	return f, name, nil
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

// LogHook sets up a logrus test hook and configures the default logger to use it.
// It returns the hook and a cleanup function to restore the default logger.
// Example usage:
//
//	hook, cleanup := LogHook()
//	defer cleanup()
//	// ... perform logging operations ...
//	Expect(hook.LastEntry()).ToNot(BeNil())
//	Expect(hook.LastEntry().Level).To(Equal(logrus.WarnLevel))
//	Expect(hook.LastEntry().Message).To(Equal("log message"))
func LogHook() (*test.Hook, func()) {
	l, hook := test.NewNullLogger()
	log.SetLevel(log.LevelWarn)
	log.SetDefaultLogger(l)
	return hook, func() {
		// Restore default logger after test
		log.SetDefaultLogger(logrus.New())
	}
}
