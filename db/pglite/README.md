# db/pglite — spike: PostgreSQL embedded in the Navidrome binary

Runs the [PGlite](https://pglite.dev) WASI build of PostgreSQL 17.5 inside the process with
[wazero](https://wazero.io) (pure Go, no CGO), and exposes it to `database/sql` through the real PostgreSQL wire
protocol. Enabled with `DbPath = "pglite://<dir>"`.

**This is exploratory. It is not a supported way to run Navidrome.** The 131 SQLite migrations are not ported, so the
schema has to be supplied by hand (`ND_PGLITE_SCHEMA`), and several queries Navidrome generates are SQLite-only.

## How it fits together

    Navidrome → database/sql → pgx → unix socket → bridge → wasm memory → PostgreSQL (wasm)

The bridge speaks no SQL. It copies wire-protocol bytes, and only inspects message framing, the handshake, and the
ReadyForQuery status byte. `PGlite.OpenDB()` swaps the unix socket for an in-process pipe; both perform the same, so
the socket is the default because external tools can use it.

## Getting the wasm binary

Not in the repository: it is a 6.7 MB archive containing a 17 MB wasm module. Build it, then point the tests at it
with `ND_PGLITE_TARBALL`, or drop it at `tmp/pglite-wasi-O2-fix.tar.gz` (the default the tests look for).

The published WASI builds are all compiled `-O0`; rebuilding with `-O2` is worth 2 to 3x on every query and cuts
startup from 16 s to 1.8 s. The build recipe and the patches, including one C fix of our own, are in the spike notes
under `tmp/pglite-build/` on the `pglite-spike` branch.

## Connecting with psql

    PGPASSWORD=x PGSSLMODE=disable psql -h /absolute/path/to/DataFolder -U postgres postgres

The exact command is logged at startup. A password is required even though PGlite skips authentication. Plain SQL
works; `\dt` and `\d` do not, because the build's session schema is `pg_catalog` and its system views are missing.

## Known limits

- **One session.** PGlite is a single PostgreSQL backend. The bridge accepts several connections and serializes them,
  holding the session across a whole transaction and a whole handshake. Four connections perform exactly like one.
- **One process.** Set `DevExternalScanner = false`. Navidrome otherwise scans in a child process, which would open a
  second PGlite on the same data directory and corrupt it.
- **Session state is shared.** `SET`, temp tables and the like leak between clients.
- **Simple protocol only.** The WASI build mishandles extended-protocol portals, so the DSN pins
  `default_query_exec_mode=simple_protocol`.
- **No `COPY FROM STDIN`.** The tick loop cannot suspend to wait for the data rows.

## Environment switches

| variable | effect |
|---|---|
| `ND_PGLITE_TARBALL` | path to the wasm archive (tests) |
| `ND_PGLITE_SCHEMA` | apply a schema file after connecting, tolerating failures |
| `ND_PGLITE_TRACE` | log wire traffic and per-tick timings |
| `ND_PGLITE_NOCMA` | force the file transport instead of shared memory |
