# db/pglite — spike: PostgreSQL embedded in the Navidrome binary

On this branch Navidrome's only database is the [PGlite](https://pglite.dev) WASI build of PostgreSQL 17.5, run
inside the process with [wazero](https://wazero.io) (pure Go, no CGO) and reached through `database/sql` and pgx over
the real PostgreSQL wire protocol. SQLite support has been removed from `db/` and the persistence layer.

**This is exploratory. It is not a supported way to run Navidrome.** It starts, runs a full scan, logs in, and
browses artists and albums through both the native and Subsonic APIs. Everything else is untested, and several
paths still carry SQLite-only SQL (see "Not ported").

## How it fits together

    Navidrome → database/sql → pgx → in-process pipe → bridge → wasm memory → PostgreSQL (wasm)

The bridge speaks no SQL. It copies wire-protocol bytes, and only inspects message framing, the handshake, and the
ReadyForQuery status byte. A unix socket is also listened on for external tools; both paths perform the same.

## Schema

The 131 SQLite migrations are gone. `db/migrations/20260905004449_initial_schema.sql` is one goose migration with
the whole schema, a crude mechanical translation of the SQLite one (`text`/`bigint`/`boolean`/`timestamp`/`jsonb`,
no FTS5, no triggers), plus the seed rows the old migrations used to insert: the default library and the default
transcodings. `DbPath` defaults to `pglite://<DataFolder>/pglite`.

## The wasm module

`pglite.wasi.gz` (4.9 MB) is embedded with `go:embed`, decompressed in memory and compiled by wazero at startup;
nothing is unpacked to disk. The compiled code is cached under the cache folder (`Config.CacheDir`), so only the
first start pays the ~1.7 s compile. The share files PostgreSQL needs are inside the module (wasi-vfs). The cluster
itself is created by `initdb` on the first start (~1.5 s, in a throwaway wasm instance) in `<DataFolder>/pglite/pgdata`;
a short-lived backend on `template1` then runs `CREATE DATABASE navidrome`, and the real backend starts on it in a
fresh instance. Later starts run PostgreSQL's real startup with WAL recovery, and every
close checkpoints through `pgl_shutdown`. `Config.WasmOverride` points at a different module file for experiments.

It is our own build: the published WASI builds are all compiled `-O0`, and rebuilding with `-O2` is worth 2 to 3x on
every query. The recipe and patches, including one C fix of our own, are in `build/`. It is built **without the wizer
snapshot** (`PGLITE_WIZER=0`): a snapshot starts faster but freezes the pristine cluster's transaction counter and WAL
position, so rows committed by an earlier run become invisible after a restart.

## Connecting with psql

    PGPASSWORD=x PGSSLMODE=disable psql -h /absolute/path/to/DataFolder -U postgres navidrome

The exact command is logged at startup. A password is required even though PGlite skips authentication. Plain SQL
works; `\dt` and `\d` do not, because the build's system views are missing.

## Known limits

- **One session.** PGlite is a single PostgreSQL backend. The bridge accepts several connections and serializes them,
  holding the session across a whole transaction and a whole handshake. Four connections perform exactly like one.
  Concurrent queries on one transaction are not possible either (pgx refuses them), so `library.RefreshStats` runs
  its queries sequentially.
- **One process.** `DevExternalScanner` is forced off: a scanner subprocess would open a second PGlite on the same
  data directory and corrupt it.
- **Session state is shared.** `SET`, temp tables and the like leak between clients.
- **Simple protocol only.** The WASI build mishandles extended-protocol portals, so the DSN pins
  `default_query_exec_mode=simple_protocol`, and untyped literals sometimes need explicit casts.
- **No `COPY FROM STDIN`**, no backup/restore (`navidrome backup` returns an error).

## Not ported (still SQLite SQL)

Full-text search (`persistence/sql_search_fts.go`, FTS5), smart-playlist criteria (`model/criteria`,
`persistence/criteria_sql.go`: `json_tree`), plugin cleanup (`persistence/plugin_cleanup.go`), `GetRandom` and
`optimizePagination` (`rowid`), natural sorting (dropped; `lower()` replaces `collate nocase`). The Go test suites
that create an in-memory SQLite database no longer run.

## Environment switches

| variable | effect |
|---|---|
| `ND_PGLITE_TRACE` | log wire traffic and per-tick timings |
| `ND_PGLITE_NOCMA` | force the file transport instead of shared memory |
