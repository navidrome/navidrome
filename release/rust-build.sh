#!/bin/sh
# Build Rust gRPC companion binaries with fat LTO (release-fat profile).
#
# Environment:
#   RUST_PROFILE          Cargo profile (default: release-fat)
#   RUSTFLAGS             extra rustc flags (e.g. -C target-cpu=znver3)
#   CARGO_BUILD_JOBS      parallel rustc jobs
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
PROFILE="${RUST_PROFILE:-release-fat}"
JOBS="${CARGO_BUILD_JOBS:-}"

cd "${ROOT}/rust"

# One-shot release builds: incremental caches add overhead without helping CI.
export CARGO_INCREMENTAL=0

# Do not fight the Cargo profile's lto=fat via RUSTFLAGS.
if [ -n "${RUSTFLAGS:-}" ]; then
  RUSTFLAGS=$(printf '%s' "${RUSTFLAGS}" | sed 's/-C lto=[^ ]*//g' | sed 's/-Clto=[^ ]*//g' | sed 's/^ *//;s/ *$//')
  export RUSTFLAGS
fi

cargo_cmd() {
  cargo "$@"
}

ARGS="build --locked --profile ${PROFILE} --bins"
if [ -n "${JOBS}" ]; then
  ARGS="${ARGS} -j ${JOBS}"
  export CARGO_BUILD_JOBS="${JOBS}"
fi

echo "[rust] building gRPC workers (profile=${PROFILE}, RUSTFLAGS=${RUSTFLAGS:-<unset>})"
# shellcheck disable=SC2086
cargo_cmd ${ARGS}
echo "[rust] binaries in ${CARGO_TARGET_DIR:-target}/${PROFILE}/"
