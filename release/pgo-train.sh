#!/bin/sh
# Collect representative CPU profiles and merge them into a Go PGO profile.
#
# Run with thin LTO enabled (see release/cgo-lto-env.sh) so profile collection
# stays fast while still exercising CGO-heavy paths.
#
# Environment:
#   GO_PGO_BENCHTIME        heavy workloads (default: 3s local, 1s in JBS)
#   GO_PGO_LIGHT_BENCHTIME  light workloads (default: 250ms local; JBS sets 100ms)
#   PGO_OUTPUT              merged profile path (default: default.pgo)
#   PGO_PROFILE_DIR         per-workload .pprof directory (default: /tmp/pgo)
#   PGO_BUILD_TAGS          go test -tags value
#   ND_SCANNERWORKERPATH    navidrome-scanner for scan benchmark (JBS sets this)
#   ND_METADATAWORKERPATH   navidrome-metadata for artwork + FTS query benchmarks
#
# Final scenario (20 workloads, overlaps removed):
#
#   Domain          | Name            | Benchmark
#   ----------------+-----------------+------------------------------------------
#   Scanner         | scan            | BenchmarkScan
#   DB / CGO sqlite | db_sqlite       | BenchmarkSQLiteHotPath
#   DB / search     | db_tags         | BenchmarkUnmarshalTags
#   DB / search     | search_fts      | BenchmarkSearchFTS5QueryCached
#   API             | api_json        | BenchmarkSubsonicJSONMarshal
#   API             | api_auth        | BenchmarkAuthUserCacheHit
#   API             | api_urls        | BenchmarkImageURL
#   API             | api_sse         | BenchmarkSSEWriteEvent
#   Streaming       | stream_decide   | BenchmarkLegacyStreamDecision
#   Streaming       | stream_cache    | BenchmarkStreamMediaCacheHit
#   Artwork         | artwork         | BenchmarkResizeFullPipeline/.../to_300
#   Compression     | compress_stream | BenchmarkCompressionReadFrom/zstd
#   HTTP/2 public   | h2_api          | BenchmarkHTTP2CompressedAPIResponse
#   HTTP/2 public   | h2_stream       | BenchmarkHTTP2StreamingResponse
#   HTTP/2 H3 bridge| h2_h3_bridge    | BenchmarkHTTP2InheritedBridgeRoundTrip
#   Public gRPC     | grpc_ping       | BenchmarkPublicGRPCPing
#   Public gRPC     | grpc_invoke     | BenchmarkPublicGRPCInvoke
#   Public gRPC     | grpc_open       | BenchmarkPublicGRPCOpenStream
#   Public gRPC     | grpc_h2_ping    | BenchmarkPublicGRPCHTTP2Ping
#   Integration     | integration_sign| BenchmarkIntegrationGatewaySign
#
# Removed as redundant with the workloads above:
#   - BenchmarkCompressionLargeSingleWrite  -> h2_api (Write compress) + compress_stream (ReadFrom)
#   - BenchmarkCopy                       -> h2_stream (pooled copy over HTTP/2)
#   - BenchmarkAuthenticatedHTTP3Bridge   -> h2_h3_bridge (auth + compress + private H2)
#   - BenchmarkToSQLArgsMediaFile           -> scan (SQLite insert during ScanAll)
set -eu

OUTPUT="${PGO_OUTPUT:-default.pgo}"
HEAVY_BENCHTIME="${GO_PGO_BENCHTIME:-3s}"
LIGHT_BENCHTIME="${GO_PGO_LIGHT_BENCHTIME:-250ms}"
PROFILE_DIR="${PGO_PROFILE_DIR:-/tmp/pgo}"

if [ -z "${PGO_BUILD_TAGS:-}" ]; then
  if [ -x ./release/build-tags.sh ]; then
    PGO_BUILD_TAGS="$(./release/build-tags.sh)"
  else
    PGO_BUILD_TAGS="netgo,sqlite_fts5"
  fi
fi

mkdir -p "$(dirname "$OUTPUT")"
mkdir -p "${PROFILE_DIR}"

PROFILE_FILES=""

train() {
  name="$1"
  package="$2"
  benchmark="$3"
  benchtime="$4"
  echo "[pgo] training ${name} (${package} ${benchmark}, benchtime=${benchtime})"
  go test \
    -run='^$' \
    -bench="${benchmark}" \
    -benchtime="${benchtime}" \
    -count=1 \
    -tags="${PGO_BUILD_TAGS}" \
    -cpuprofile="${PROFILE_DIR}/${name}.pprof" \
    "${package}"
  test -s "${PROFILE_DIR}/${name}.pprof"
  PROFILE_FILES="${PROFILE_FILES} ${PROFILE_DIR}/${name}.pprof"
}

echo "[pgo] starting training: heavy=${HEAVY_BENCHTIME} light=${LIGHT_BENCHTIME}"

# Scanner (CGO + SQLite + optional Rust worker)
train scan ./scanner '^BenchmarkScan$' "${HEAVY_BENCHTIME}"

# CGO sqlite amalgamation hot path (WAL upsert/select via production driver)
train db_sqlite ./db '^BenchmarkSQLiteHotPath$' "${HEAVY_BENCHTIME}"

# DB reads and search query preparation (Go-side; pairs with db_sqlite for CGO)
train db_tags ./persistence '^BenchmarkUnmarshalTags$' "${HEAVY_BENCHTIME}"
train search_fts ./persistence '^BenchmarkSearchFTS5QueryCached$' "${LIGHT_BENCHTIME}"

# Subsonic / native API hot paths
train api_json ./server/subsonic '^BenchmarkSubsonicJSONMarshal$' "${HEAVY_BENCHTIME}"
train api_auth ./server/subsonic '^BenchmarkAuthUserCacheHit$' "${LIGHT_BENCHTIME}"
train api_urls ./core/publicurl '^BenchmarkImageURL$' "${LIGHT_BENCHTIME}"
train api_sse ./server/events '^BenchmarkSSEWriteEvent$' "${LIGHT_BENCHTIME}"

# Playback streaming
train stream_decide ./core/stream '^BenchmarkLegacyStreamDecision$' "${LIGHT_BENCHTIME}"
train stream_cache ./server/subsonic '^BenchmarkStreamMediaCacheHit$' "${LIGHT_BENCHTIME}"

# Cover art (optional Rust metadata worker)
train artwork ./core/artwork '^BenchmarkResizeFullPipeline/jpeg/1000x1000_to_300$' "${HEAVY_BENCHTIME}"

# Adaptive compression on streaming response bodies (ReadFrom path)
train compress_stream ./server '^BenchmarkCompressionReadFrom/zstd$' "${HEAVY_BENCHTIME}"

# Public TLS HTTP/2 (replaces isolated compression Write + pooled copy micro-bench)
train h2_api ./server '^BenchmarkHTTP2CompressedAPIResponse$' "${HEAVY_BENCHTIME}"
train h2_stream ./server '^BenchmarkHTTP2StreamingResponse$' "${LIGHT_BENCHTIME}"

# H3 companion private HTTP/2 bridge (auth middleware + compress + framing)
train h2_h3_bridge ./server '^BenchmarkHTTP2InheritedBridgeRoundTrip$' "${HEAVY_BENCHTIME}"

# Public gRPC (in-process bufconn + TLS HTTP/2 mux framing)
train grpc_ping ./server/publicgrpc '^BenchmarkPublicGRPCPing$' "${LIGHT_BENCHTIME}"
train grpc_invoke ./server/publicgrpc '^BenchmarkPublicGRPCInvoke$' "${HEAVY_BENCHTIME}"
train grpc_open ./server/publicgrpc '^BenchmarkPublicGRPCOpenStream$' "${HEAVY_BENCHTIME}"
train grpc_h2_ping ./server/publicgrpc '^BenchmarkPublicGRPCHTTP2Ping$' "${LIGHT_BENCHTIME}"

# Outbound integration gateway (gRPC worker when ND_INTEGRATIONWORKERPATH is set)
train integration_sign ./core/integration '^BenchmarkIntegrationGatewaySign$' "${LIGHT_BENCHTIME}"

echo "[pgo] merging $(echo "${PROFILE_FILES}" | wc -w | tr -d ' ') profiles"
go tool pprof \
  -proto \
  -output="${OUTPUT}" \
  ${PROFILE_FILES}

test -s "${OUTPUT}"
echo "[pgo] merged profile written to ${OUTPUT} ($(wc -c <"${OUTPUT}") bytes)"
go tool pprof -top -nodecount=15 "${OUTPUT}" || true
