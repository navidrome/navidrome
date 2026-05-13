package conf

import (
	"fmt"
	"os"
	"sync"
)

// Dir wraps a directory path and lazily creates the directory on first use.
type Dir struct {
	path string
	once sync.Once
	err  error
}

// NewDir creates a new Dir with the given path. No side effects.
func NewDir(path string) Dir {
	return Dir{path: path}
}

// String returns the raw path without creating the directory. Satisfies fmt.Stringer.
func (d *Dir) String() string {
	return d.path
}

// Path creates the directory on first call (via sync.Once) and returns the path.
func (d *Dir) Path() (string, error) {
	d.once.Do(func() {
		if d.path == "" {
			return
		}
		d.err = os.MkdirAll(d.path, 0700)
		if d.err != nil {
			d.err = fmt.Errorf("creating directory %q: %w", d.path, d.err)
		}
	})
	return d.path, d.err
}

// MustPath calls Path() and calls logFatal on error.
func (d *Dir) MustPath() string {
	path, err := d.Path()
	if err != nil {
		logFatal(err)
	}
	return path
}

// MarshalText returns the raw path bytes. No side effects.
func (d *Dir) MarshalText() ([]byte, error) {
	return []byte(d.path), nil
}

// UnmarshalText sets the path from bytes. No side effects.
func (d *Dir) UnmarshalText(text []byte) error {
	d.path = string(text)
	return nil
}
