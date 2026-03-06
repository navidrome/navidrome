//go:build sqlite_spellfix

package buildtags

// SPELLFIX is required for the spellfix1 virtual table, used for fuzzy/approximate
// string matching. Without this tag, the SQLite driver won't include spellfix1 support.

var SQLITE_SPELLFIX = true
