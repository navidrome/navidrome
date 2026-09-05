# Rebuilding PGlite for WASI with optimization (2026-09-04)

Source of the build: https://github.com/roastedroot/pglite4j `wasm-build/` (self-contained Docker build of
electric-sql/postgres-pglite, branch REL_17_5_WASM-pglite). Clone it to /tmp/pglite4j and apply the diffs here:

- `Dockerfile-arm64.diff` — the image downloads x86_64 toolchains; on Apple Silicon Docker (arm64) clang cannot run.
  Swaps wasi-sdk, binaryen, wasmtime, wizer and wasi-vfs for their arm64/aarch64 assets.
- `wasm-build.sh.diff` — upstream overwrites `CC_PGLITE`, and PostgreSQL's configure does not add its default `-O2` when
  `CFLAGS` is given, so the stock build is `-O0`. Injects `${PGLITE_OPT:--O2}` at the front of `CC_PGLITE`.
- `build.sh.diff` — passes `PGLITE_OPT` (and `PGLITE_PIC`, unused so far) into the container.
- `wasi-c.diff` — makes the forced `-fPIC` in the compiler wrapper overridable (`PGLITE_PIC`). NOTE: the matching
  `-fpic` in `postgresql-src/wasm-build/build-pgcore.sh` (cloned at build time) is not covered by a diff here; the
  `-fno-pic` experiment did not take effect and is unmeasured.
- `clear_error_portals.diff` — goes in `patches/pglite-wasm/`. A PostgreSQL ERROR traps the wasm instance, which skips
  every PG_CATCH, so the active portal is never marked failed and the next statement fails with
  "cannot drop active portal". Wrapping `AbortCurrentTransaction()` in `clear_error()` with
  `shmem_exit_inprogress = true` makes `AtAbort_Portals` mark it failed.

Build (about 5 minutes on 16 cores):

    cd /tmp/pglite4j/wasm-build
    CLEAN=true DEBUG=false PGLITE_OPT=-O2 ./build.sh
    xz -dc output/sdk-dist/pglite-wasi.tar.xz | gzip -1 > pglite-wasi-O2-fix.tar.gz

`DEBUG=false` matters: `DEBUG=true` (the default) builds single-threaded and keeps debug output.

Output layout ("v2" in db/pglite): `tmp/pglite/bin/pglite.wasi` with the share files embedded through wasi-vfs and a
pre-initialized cluster in `pgdata/` produced by wizer, so `initdb` never runs at startup (1.8 s instead of 13 s).
Preopens must be `/tmp`, `/pgdata`, `/dev` in that order; `PGDATA=/pgdata`; do not call `_start`/`pgl_initdb`/
`pgl_backend`; call `interactive_write(0)` once; on a trap check `pgl_check_error()` then `clear_error()`,
`interactive_write(-1)`, one more `interactive_one()`.

Result archive used by the tests and benchmark: `tmp/pglite-wasi-O2-fix.tar.gz` (6.7 MB; wasm 17.3 MB).

## `-fpic`: measured, no effect (do not redo)

`PGLITE_PIC` must be patched into **three** places, and the one that matters is easy to miss:
`wasm-build/build-pgcore.sh` at the workspace root is the copy that runs — `postgresql-src/wasm-build/build-pgcore.sh`
is not. (Same for `wasm-build.sh`.) With all three patched, `-fno-pic` reaches configure and every compile.

Result: no measurable difference on any query, and the final wasm code section changes by 96 bytes out of 5.35 MB.
`-fPIC` does change object files, but `wasm-ld` relaxes the relocations back to direct addressing when producing a
static executable. Keep the upstream default.

## Without the wizer snapshot (the durable build; 2026-09-04)

The snapshot makes reopening a data dir unsafe: it carries the pristine cluster's transaction counter and WAL
position, and `pgl_backend` traps if asked to restart on a live snapshot. Build with `PGLITE_WIZER=0` instead:

    cd /tmp/pglite4j/wasm-build
    DEBUG=false PGLITE_OPT=-O2 PGLITE_WIZER=0 ./build.sh
    # package with a marker the bridge recognizes:
    tar xJf output/sdk-dist/pglite-wasi.tar.xz -C /some/dir && touch /some/dir/tmp/pglite/NOSNAPSHOT
    (cd /some/dir && tar czf pglite-wasi.tar.gz tmp pgdata)

Two quirks. The initdb pre-population step (`wasmtime run --env INITDB_ONLY=1`) failed on a clean `output/pgdata`
with "cannot make new WAL entries during recovery"; restoring a `pgdata/` from an earlier good archive into
`output/pgdata` before building sidesteps it (the build then skips pre-init). And the cluster only has `template1`;
the bridge connects to it explicitly. `build.sh.diff` now wraps only the wizer invocation in `PGLITE_WIZER`.
