//go:build sqlite_fts5

package buildtags

// NOTICE: This file was created to force the inclusion of the `sqlite_fts5` tag when compiling the project.
// If the tag is not included, the compilation will fail because this variable won't be defined, and the `main.go`
// file requires it.

// Why this tag is required? FTS5 is used for full-text search. Without it, the SQLite driver won't include
// FTS5 support and the application will fail at runtime when running migrations or executing search queries.

var SQLITE_FTS5 = true
