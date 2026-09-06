# Navidrome Rust search worker

This crate is the incremental search engine used to move relevance ranking and
substring search off SQLite's broad `LIKE` fallback. It keeps a Tantivy index
behind gRPC/Protobuf (`--grpc-worker`). NDJSON stdin/stdout remains available in the
binary for manual debugging only; Go never uses it.

The index combines exact-name ranking with Unicode 2–3 character n-grams. This
makes short Korean, Japanese, and Chinese names searchable without scanning all
rows, while retaining weighted primary and secondary fields for Latin text.

The protocol supports atomic chunked replacement, incremental upserts/deletes,
scoped queries, and stats. A replacement keeps the previous reader visible until
the final commit, so large libraries never expose a partial index. After the
initial snapshot, Go applies scan deltas through `upsert`/`delete` and falls
back to a full replacement when the delta is large or document counts drift.
Until the Go synchronization layer has completed an initial snapshot, SQLite
FTS5 remains the authoritative fallback.
