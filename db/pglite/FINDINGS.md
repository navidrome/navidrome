# Embedded PostgreSQL spike: findings

Branch `pglite-spike`, PR #6084. Everything below was measured on the PGlite WASI build of PostgreSQL 17.5
running inside the Navidrome process under wazero, against a 97,592-track library (7,019 albums, 29,602
artists, 1.9 TB of audio) on a QNAP NAS (x86_64, slower than a laptop), plus a laptop for micro-benchmarks.

## What works

- Navidrome starts, creates the cluster, runs a full scan, imports playlists, logs in, browses, searches,
  plays, scrobbles, and survives restarts. The browser UI and the Subsonic API were exercised page by page
  and endpoint by endpoint (`tmp/hammer` cycles about 90 calls per iteration) with zero failures on an idle
  server after the last fixes.
- The wasm module is embedded in the binary (`pglite.wasi.gz`, 4.9 MB), decompressed in memory, compiled by
  wazero with a disk cache. Nothing of it is written to disk. The cluster is created by `initdb` at runtime
  on the first start (about 1.5 s) and lives in `<DataFolder>/pglite/pgdata`.
- Only three host directories are visible to the module: the cluster, a scratch dir mounted as `/tmp`, and a
  fake `/dev/urandom`. There is no network.

## Numbers

| measurement | value |
|---|---|
| full scan from scratch, 97,592 tracks | 32m58s (phase 1: 30m26s, about 3,300 tracks/min) |
| same scan while the API was hammered continuously | 1h10m (about 600 tracks/min) |
| full re-scan, nothing changed | 2m46s |
| one-folder quick scan | under 1 s of work plus about 26 s of fixed post-scan steps |
| artist stats refresh, 29,602 artists | 1m21s (one statement, 3.3 s on the laptop) |
| tag purge / tag counts / vacuum+analyze / checkpoint | 8 s / 5 s / 12 s / 2 s |
| memory during a scan | 600 to 950 MB resident |
| database size | 628 MB (SQLite on the same library: 729 MB), plus 130 to 150 MB of write-ahead log |
| cold start (compile + initdb + create database) | about 5 s; warm start about 0.2 s |
| UI list requests on an idle server | 30 to 300 ms; artist list 0.2 s |

## What we learned

**The backend is one session, and everything follows from that.** PostgreSQL's concurrency comes from one
process per connection sharing memory, and wasm has neither fork nor shared memory between instances. PGlite
therefore runs PostgreSQL's single-user mode. The bridge accepts several client connections and serializes
them onto that one session, holding it across a whole transaction and a whole handshake. Consequences:

- Four pooled connections perform exactly like one.
- During a scan, every UI request waits behind the scanner's current transaction. The scanner holds a
  transaction about 87% of its run.
- Hammering the API during a scan does not break anything, but slows the import fivefold and makes requests
  queue for tens of seconds.
- A second wasm instance on the same cluster is not an option: each has a private page cache.
- PGlite's own multiplexer (`@electric-sql/pglite-socket`, TypeScript) is the same idea and has the same
  limits. The maintainer's [threaded PostgreSQL experiment](https://github.com/samwillis/multithreaded-postgres)
  is what would lift this, and it is research code today.

**The shipped wasm builds are `-O0`.** Rebuilding with `-O2` (no wizer snapshot) is 2 to 3 times faster on
every query. Recipe and patches in `build/`.

**PGlite's fork had a per-message memory leak.** PostgreSQL 17 removed `MemoryContextResetAndDeleteChildren`;
the fork stubbed it as an empty macro, so `MessageContext` was never reset and every wire message leaked its
parse and plan state, about 10 KB. The first real scan drove the process to 15 GB and the NAS into swap.
Patched in the build (`build/message_context_reset.diff`).

**PGlite does not read config files.** Its restart path skips `SelectConfigFiles`, `ALTER SYSTEM` writes to
the wrong directory, and `pg_reload_conf()` is a no-op without a postmaster. Settings come from a hardcoded
`-c` list in the fork (`WASM_PGOPTS`). `max_wal_size` was added there (`build/max_wal_size.diff`), which caps
the log directory near 130 MB instead of 1 GB.

**PGlite's single-user loop drops requests when a client disappears mid-handshake.** A pool connection attempt
that timed out between the startup packet and the password step left the backend in
`ClientAuthInProgress`; from then on it answered other clients without `ReadyForQuery`, or read their query
as an empty one, until the next handshake completed. Each such reply killed a pooled connection and the
scanner transaction on it, and the UI bounced back to the login page. The bridge now answers every handshake
after the first one itself, from the cached `ParameterStatus` messages, so the backend never sees another
authentication.

**No autovacuum, no checkpointer.** In single-user mode nothing runs in the background. Every stats refresh
leaves a dead version of all 29,602 `library_artist` rows, and nothing is checkpointed until shutdown. The
post-scan step now runs `VACUUM (ANALYZE)` and `CHECKPOINT`.

**The planner needs statistics, and index-friendly predicates.** Without `ANALYZE` the row estimates are off
by orders of magnitude; the first scan runs with no statistics at all. Even with them, a role filter written
as `stats->'role'->>'m' IS NOT NULL` made the artist list probe 29,602 rows one by one (116 s). A GIN index
on the jsonb column and a containment predicate (`stats @> '{"role": {}}'`) bring it to 0.2 s.

**SQL translated from SQLite needs a second look for scale, not just syntax.**

- `NOT IN (subquery)` over a million-row subquery re-evaluates the subquery per row (55 s at scale);
  `id IN (SELECT id FROM t EXCEPT ...)` takes 1 s.
- A CTE referenced once is inlined; a correlated subquery in an `UPDATE` then re-runs the whole aggregation
  per row. `AS MATERIALIZED` plus a join fixed the artist stats (minutes per batch to 0.2 s).
- `integer` columns that hold byte totals overflow (`library.total_size` for a 2 TB library); use `bigint`.
- `[]byte` query arguments become `bytea` under pgx; JSON stored in text columns must be bound as strings.
- `COALESCE(bool, 0)`, `SELECT DISTINCT ... ORDER BY random()`, `rowid`, and the reserved word `user` all
  needed rewrites.

**The bridge must never leave a client without a reply.** Two hangs came from the bridge itself: a five
second write deadline that cut slow readers mid-message (phase 3 streams all albums through a cursor while
running queries per row), and forwarding a reply without `ReadyForQuery` as if it were complete. Replies are
now sent without a deadline, incomplete ones get a synthesized error plus `ReadyForQuery`, and a request the
backend produced no output for is handed over again.

**`database/sql` churns connections by default.** With `MaxOpenConns(4)` and the default of two idle
connections, the pool opened and closed a connection about 225 times a minute during a scan. Every open is a
handshake through the single session. `SetMaxIdleConns(4)` stopped it.

## Fixes on this branch

Bridge and build:

- memory leak in the wasm module (`message_context_reset.diff`)
- `max_wal_size` cap in the wasm module
- handshakes answered by the bridge; `SSLRequest` answered with `N`; `CancelRequest` dropped
- no write deadline on replies; incomplete replies fail instead of hanging; empty replies are resent
- pooled connections kept idle instead of reopened
- dedicated scratch directory as the guest `/tmp`
- `VACUUM (ANALYZE)` and `CHECKPOINT` after each scan

Schema and SQL:

- one goose migration with the whole schema; `user` renamed to `user_account`; `bigint` for byte totals;
  `bigserial` for `scrobbles.id`; GIN index on `library_artist.stats`
- artist stats: materialized CTEs, one join, one statement for a full refresh
- tag purge with `EXCEPT`; `has_rating` and `starred` filters typed correctly; playlist rules bound as text;
  playlist cover query with `GROUP BY`; random songs by id; counts through a subquery; share and library
  queries on `user_account`; artwork queue upsert with `GREATEST`
- search defaults to the LIKE backend (FTS5 is SQLite-only)

## Limitations

- One session. UI and scanner take turns; concurrent readers do not exist.
- Not ported: full-text search (LIKE is used), smart playlist criteria (`json_tree`), backup and restore,
  natural sort. The Go suites that build an in-memory SQLite database do not run on this branch.
- `DevExternalScanner` is forced off: a scanner subprocess would open a second cluster instance.
- CPU-heavy SQL is 2 to 3 times slower than SQLite even at `-O2`; the artist stats refresh is 25 times
  slower on the NAS than on a laptop.
- The post-scan steps (tag purge, tag counts, vacuum) add about 25 s to any scan, even a one-folder one.
- Session state (`SET`, temp tables) is shared by all clients. Only the simple protocol is used.
- PGlite's WASI build is labelled experimental upstream. We carry three C patches of our own.

## Ideas for improvements

Cheap, and worth doing regardless:

- Shorter scanner transactions. This is the only lever on UI stalls during scans.
- Run the vacuum only after full scans or when a scan changed many rows; skip the fixed post-scan steps for
  one-folder scans.
- Run `ANALYZE` after phase 1 of the first scan, so the later phases plan with statistics.
- Scanner subprocess and CLI commands: connect to the parent's unix socket when a server is up, else start
  the wasm; an exclusive lock file on `pglite/host.lock` guarantees a single host. Designed, not built.
- A bridge scheduler that lets UI requests in between the scanner's transactions instead of first come,
  first served.
- Narrow the scratch mount to the two files the guest needs.

Bigger:

- Port smart playlist criteria and a real full-text search (`tsvector`, or `pg_trgm` if the build carries
  it).
- Watch the threaded PostgreSQL work. When PGlite ships a multi-session build, everything on this branch
  (schema, SQL, bridge protocol handling, operational fixes) carries over; only the session lock goes away.
- Upstream the memory leak fix and the handshake behaviour to PGlite; they have open issues with the same
  symptoms (#985, #1046).

## Tools left in `tmp/` (not committed)

`hammer` (endpoint cycler), `apicheck` (17 API checks), `leak` (wasm memory per statement), `notin` and
`stats` (query benchmarks at scale), `phase3` (cursor plus concurrent writers), `wal` (checkpoint and log
size), `run-pg.sh` (local run/reset).
