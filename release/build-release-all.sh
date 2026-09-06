#!/bin/sh
# Build Navidrome with maximum release optimizations for Go + Rust gRPC workers.
#
# Go: thin LTO PGO training -> fat LTO + merged profile
# Rust: LLVM PGO (when llvm-profdata available) + release-fat, else fat LTO only
#
# Environment mirrors make build-release / build-rust-workers variables.
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

chmod +x ./release/rust-build.sh ./release/rust-pgo-train.sh ./release/pgo-train.sh ./release/cgo-lto-env.sh ./release/sqlite-cflags.sh

if [ "${RUST_PGO_ENABLED:-true}" = "true" ] && command -v llvm-profdata >/dev/null 2>&1; then
  echo "[release-all] Rust PGO enabled (llvm-profdata found)"
  ./release/rust-pgo-train.sh
else
  echo "[release-all] Rust PGO skipped; using fat LTO only"
  ./release/rust-build.sh
fi

export CGO_ENABLED=1
# Prefer clang+lld for CGO fat LTO (same as Dockerfile.jbs gobuilder).
if command -v clang >/dev/null 2>&1; then
  export CC="${CC:-clang}"
  export CXX="${CXX:-clang++}"
fi
if [ "$(uname -m)" = "x86_64" ] && [ -z "${GOAMD64:-}" ]; then
  export GOAMD64=v4
fi

if [ "${GO_PGO_ENABLED:-true}" = "true" ]; then
  echo "[release-all] collecting Go PGO profile"
  eval "$(./release/cgo-lto-env.sh thin)"
  PGO_BUILD_TAGS="${PGO_BUILD_TAGS:-$(./release/build-tags.sh 2>/dev/null || echo netgo,sqlite_fts5)}" \
    GO_PGO_BENCHTIME="${GO_PGO_BENCHTIME:-3s}" \
    PGO_OUTPUT="${PGO_OUTPUT:-default.pgo}" \
    ./release/pgo-train.sh
  PGO_FLAG="-pgo=${PGO_OUTPUT:-default.pgo}"
else
  PGO_FLAG="-pgo=off"
fi

eval "$(./release/cgo-lto-env.sh fat)"
TAGS="${PGO_BUILD_TAGS:-$(./release/build-tags.sh 2>/dev/null || echo netgo,sqlite_fts5)}"
GIT_SHA="${GIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo dev)}"
GIT_TAG="${GIT_TAG:-$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)-SNAPSHOT}"

echo "[release-all] linking Go binary (fat LTO + PGO)"
go build \
  ${PGO_FLAG} \
  -trimpath \
  -buildvcs=false \
  -ldflags="-w -s -X github.com/navidrome/navidrome/consts.gitSha=${GIT_SHA} -X github.com/navidrome/navidrome/consts.gitTag=${GIT_TAG}" \
  -tags="${TAGS}"

echo "[release-all] done"
