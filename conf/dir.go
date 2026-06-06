package conf

import (
	"cmp"
	"fmt"
	"os"
)

// Dir wraps a directory path and creates the directory on demand. Dir is a
// plain value type — safe to copy, compare, and print via reflection-based
// formatters (pretty.Sprintf("%# v", ...)) without any concurrency hazards.
// Directory creation is delegated to os.MkdirAll on every Path() call;
// MkdirAll is idempotent, so repeated calls cost one stat syscall when the
// directory already exists.
type Dir struct {
	path string
	perm os.FileMode
}

// NewDir creates a new Dir with the given path and default permissions (os.ModePerm).
func NewDir(path string) Dir {
	return Dir{path: path, perm: os.ModePerm}
}

// NewDirWithPerm creates a new Dir with the given path and permissions.
// A perm of 0 is treated as "default" and resolves to os.ModePerm at
// directory-creation time; pass an explicit non-zero mode to constrain the
// permissions.
func NewDirWithPerm(path string, perm os.FileMode) Dir {
	return Dir{path: path, perm: perm}
}

// String returns the raw path without creating the directory. Satisfies fmt.Stringer.
func (d Dir) String() string {
	return d.path
}

// Path ensures the directory exists and returns its path. Safe to call
// repeatedly; an empty path is returned as-is with no error.
func (d Dir) Path() (string, error) {
	if d.path == "" {
		return "", nil
	}
	if err := os.MkdirAll(d.path, cmp.Or(d.perm, os.ModePerm)); err != nil {
		return d.path, fmt.Errorf("creating directory %q: %w", d.path, err)
	}
	return d.path, nil
}

// MustPath calls Path() and calls logFatal on error.
func (d Dir) MustPath() string {
	path, err := d.Path()
	if err != nil {
		logFatal("creating directory:", err)
	}
	return path
}

// GoString implements fmt.GoStringer so that %#v (used by pretty.Sprintf)
// prints the path string instead of the internal struct fields.
func (d Dir) GoString() string {
	return fmt.Sprintf("%q", d.path)
}

// MarshalText returns the raw path bytes. No side effects.
func (d Dir) MarshalText() ([]byte, error) {
	return []byte(d.path), nil
}

// UnmarshalText sets the path from bytes. No side effects.
func (d *Dir) UnmarshalText(text []byte) error {
	d.path = string(text)
	if d.perm == 0 {
		d.perm = os.ModePerm
	}
	return nil
}
