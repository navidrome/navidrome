#!/bin/sh
# Profile-guided optimization for Rust gRPC workers.
#
# 1. Build hot-path workspace binaries with -Cprofile-generate (release-pgo-gen:
#    LTO off, many codegen-units — fast instrument builds)
# 2. Run criterion benches (integration, metadata, scanner, search)
# 3. Optionally run Go integration gRPC tests against the instrumented worker
# 4. llvm-profdata merge -> merged.profdata
# 5. Rebuild all production workers with -Cprofile-use + release-fat LTO in one
#    link (apikeys has no dedicated profile — expected LLVM missing-function
#    warnings on that crate and cold deps; splitting would redo fat LTO)
#
# Environment:
#   RUST_PGO_DIR            profile output directory (default: /tmp/rust-pgo)
#   RUST_PGO_BENCH_TIME     criterion measurement time in seconds (default: 3)
#   RUST_PGO_GO_ROUNDS      Go integration test repetitions (default: 25, 0=skip)
#   RUST_PROFILE            final Cargo profile (default: release-fat)
#   RUST_PGO_GEN_PROFILE    instrument Cargo profile (default: release-pgo-gen)
#   RUST_PGO_BINS           space-separated bins that receive profile-use
#                           (default: metadata scanner search integration)
#   RUST_FAT_ONLY_BINS      extra bins linked in the final fat+PGO build without
#                           dedicated training (default: navidrome-apikeys)
#   CARGO_TARGET_DIR        cargo output directory
#   CARGO_BUILD_JOBS        parallel rustc jobs
#   RUSTFLAGS               extra rustc flags (e.g. -C target-cpu=znver3)
#   RUST_PGO_HOST_CPU       CPU for instrument+bench phases (default: x86-64)
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
PGO_DIR="${RUST_PGO_DIR:-/tmp/rust-pgo}"
BENCH_TIME="${RUST_PGO_BENCH_TIME:-3}"
GO_ROUNDS="${RUST_PGO_GO_ROUNDS:-25}"
PROFILE="${RUST_PROFILE:-release-fat}"
GEN_PROFILE="${RUST_PGO_GEN_PROFILE:-release-pgo-gen}"
PGO_HOST_CPU="${RUST_PGO_HOST_CPU:-x86-64}"
MERGED="${PGO_DIR}/merged.profdata"
TARGET_DIR="${CARGO_TARGET_DIR:-${ROOT}/rust/target}"
INSTR_TARGET="${TARGET_DIR}/pgo-instrument"
FINAL_TARGET="${TARGET_DIR}/pgo-final"
JOBS="${CARGO_BUILD_JOBS:-}"

# Workers exercised by criterion benches / Go PGO integration tests.
PGO_BINS="${RUST_PGO_BINS:-navidrome-metadata navidrome-scanner navidrome-search navidrome-integration}"
# No dedicated hot-path profile today; still ship with fat LTO.
FAT_ONLY_BINS="${RUST_FAT_ONLY_BINS:-navidrome-apikeys}"

case "${BENCH_TIME}" in
  *s) BENCH_TIME="${BENCH_TIME%s}" ;;
esac

cargo_cmd() {
  # Prefer CARGO_BUILD_JOBS (exported below). Do not pass `cargo -j` (not a
  # global option on all cargo versions) and never put -j after "--" (Criterion).
  cargo "$@"
}

bin_args() {
  _args=""
  for _b in $1; do
    _args="${_args} --bin ${_b}"
  done
  printf '%s' "${_args}"
}

# Instrumentation and profile collection run on the build host. Release
# RUSTFLAGS may target znver3 (or another micro-arch), but CI runners are not
# guaranteed to execute those instructions (SIGILL during criterion benches).
pgo_host_rustflags() {
  _base="${RUSTFLAGS:-}"
  _base=$(printf '%s' "$_base" | sed 's/-C target-cpu=[^ ]*//g' | sed 's/^ *//;s/ *$//')
  # Strip any caller LTO flags; the generate profile owns LTO=off.
  _base=$(printf '%s' "$_base" | sed 's/-C lto=[^ ]*//g' | sed 's/-Clto=[^ ]*//g' | sed 's/^ *//;s/ *$//')
  if [ -n "$_base" ]; then
    printf '%s -C target-cpu=%s' "$_base" "$PGO_HOST_CPU"
  else
    printf '%s' "-C target-cpu=${PGO_HOST_CPU}"
  fi
}

# Final profile-use flags: keep caller RUSTFLAGS (target-cpu) but never override
# Cargo release-fat's lto=fat via -C lto=... (would fight the profile).
pgo_use_rustflags() {
  _base="${RUSTFLAGS:-}"
  _base=$(printf '%s' "$_base" | sed 's/-C lto=[^ ]*//g' | sed 's/-Clto=[^ ]*//g' | sed 's/^ *//;s/ *$//')
  printf '%s' "${_base}"
}

HOST_RUSTFLAGS="$(pgo_host_rustflags)"
USE_BASE="$(pgo_use_rustflags)"

if ! command -v llvm-profdata >/dev/null 2>&1; then
  echo "[rust-pgo] llvm-profdata not found; install LLVM tools (clang/llvm)" >&2
  exit 1
fi

mkdir -p "${PGO_DIR}"
rm -f "${PGO_DIR}"/*.profraw "${MERGED}"

# profile-generate + LTO off (also set in release-pgo-gen). codegen-units come
# from the Cargo profile so instrument compiles stay parallel and fast.
GEN_FLAGS="${HOST_RUSTFLAGS} -Cprofile-generate=${PGO_DIR}"

export CARGO_INCREMENTAL=0
if [ -n "${JOBS}" ]; then
  export CARGO_BUILD_JOBS="${JOBS}"
fi

PGO_BIN_ARGS="$(bin_args "${PGO_BINS}")"

echo "[rust-pgo] phase 1: instrumented build (profile=${GEN_PROFILE}, bins=${PGO_BINS}, host cpu=${PGO_HOST_CPU})"
(
  cd "${ROOT}/rust"
  # shellcheck disable=SC2086
  CARGO_TARGET_DIR="${INSTR_TARGET}" \
    RUSTFLAGS="${GEN_FLAGS}" \
    cargo_cmd build --locked --profile "${GEN_PROFILE}" ${PGO_BIN_ARGS}
)

echo "[rust-pgo] phase 2: collecting profiles (criterion ${BENCH_TIME}s)"
for bench in integration metadata scanner search; do
  bench_dir="${ROOT}/rust/benchmarks/${bench}"
  if [ ! -f "${bench_dir}/Cargo.toml" ]; then
    echo "[rust-pgo] skip missing bench crate ${bench}"
    continue
  fi
  bench_name="${bench}_hotpaths"
  echo "[rust-pgo]   bench ${bench}"
  (
    cd "${bench_dir}"
    CARGO_TARGET_DIR="${INSTR_TARGET}" \
      RUSTFLAGS="${GEN_FLAGS}" \
      cargo_cmd bench --profile "${GEN_PROFILE}" --locked --bench "${bench_name}" -- --measurement-time "${BENCH_TIME}"
  )
done

# Instrumented binary path: custom profiles still land under target/<profile>/.
INTEGRATION_BIN="${INSTR_TARGET}/${GEN_PROFILE}/navidrome-integration"
if [ ! -x "${INTEGRATION_BIN}" ]; then
  # Older cargo layouts / fallback if profile dir naming differs.
  INTEGRATION_BIN="${INSTR_TARGET}/release/navidrome-integration"
fi
if [ "${GO_ROUNDS}" != "0" ] && [ -x "${INTEGRATION_BIN}" ] && command -v go >/dev/null 2>&1; then
  echo "[rust-pgo] phase 2b: Go gRPC integration tests (${GO_ROUNDS} rounds)"
  (
    cd "${ROOT}"
    i=1
    while [ "${i}" -le "${GO_ROUNDS}" ]; do
      ND_INTEGRATIONWORKERPATH="${INTEGRATION_BIN}" \
      ND_GRPCWORKERINTESTS=1 \
        go test -tags netgo,sqlite_fts5 -run '^TestIntegrationWorkerGRPC$' -count=1 ./core/integration/ >/dev/null
      i=$((i + 1))
    done
  )
fi

PROFRAW_COUNT="$(find "${PGO_DIR}" -name '*.profraw' 2>/dev/null | wc -l | tr -d ' ')"
if [ "${PROFRAW_COUNT}" = "0" ]; then
  echo "[rust-pgo] no .profraw files collected in ${PGO_DIR}" >&2
  exit 1
fi

echo "[rust-pgo] phase 3: merge ${PROFRAW_COUNT} profile(s)"
llvm-profdata merge -o "${MERGED}" "${PGO_DIR}"/*.profraw
test -s "${MERGED}"

# Missing-function warnings on cold transitive crates (e.g. unused tonic paths)
# are expected; we only train hot workers. Keep the flag for CI visibility.
if [ -n "${USE_BASE}" ]; then
  USE_FLAGS="${USE_BASE} -Cprofile-use=${MERGED} -Cllvm-args=-pgo-warn-missing-function"
else
  USE_FLAGS="-Cprofile-use=${MERGED} -Cllvm-args=-pgo-warn-missing-function"
fi

# One fat-LTO link for every production worker. Instrument only covered hot
# bins (phase 1); cold bins still get fat LTO here with sparse/empty profiles.
FINAL_BIN_ARGS="$(bin_args "${PGO_BINS} ${FAT_ONLY_BINS}")"
echo "[rust-pgo] phase 4: fat LTO + profile-use (profile=${PROFILE}, bins=${PGO_BINS} ${FAT_ONLY_BINS})"
(
  cd "${ROOT}/rust"
  # shellcheck disable=SC2086
  CARGO_TARGET_DIR="${FINAL_TARGET}" \
    RUSTFLAGS="${USE_FLAGS}" \
    cargo_cmd build --locked --profile "${PROFILE}" ${FINAL_BIN_ARGS}
)

echo "[rust-pgo] done: ${MERGED} ($(wc -c <"${MERGED}") bytes)"
echo "[rust-pgo] binaries: ${FINAL_TARGET}/${PROFILE}/"
