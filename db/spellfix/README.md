# spellfix1 SQLite Extension

This package statically compiles the [spellfix1](https://sqlite.org/spellfix1.html) SQLite extension
into the Navidrome binary. It is registered via `sqlite3_auto_extension` so that every new SQLite
connection has `spellfix1` available without loading a shared library.

## Vendored Files

The C source files are vendored because cgo cannot reference headers from other Go modules:

- **`spellfix.c`** — from the SQLite source tree: [`ext/misc/spellfix.c`](https://github.com/sqlite/sqlite/blob/master/ext/misc/spellfix.c)
- **`sqlite3ext.h`** — from the SQLite source tree: [`src/sqlite3ext.h`](https://github.com/sqlite/sqlite/blob/master/src/sqlite3ext.h)

## Updating

When upgrading `github.com/mattn/go-sqlite3`, run the update script to download
the matching spellfix1 source files for the bundled SQLite version:

```bash
./db/spellfix/update.sh
```

The script reads the SQLite version from go-sqlite3's `sqlite3-binding.h` and
downloads the corresponding files from the [SQLite GitHub mirror](https://github.com/sqlite/sqlite).
