//go:build sqlite_fts5

package buildtags

// FTS5 is required for full-text search. Without this tag, the SQLite driver
// won't include FTS5 support, causing runtime failures on migrations and search queries.

var SQLITE_FTS5 = true
