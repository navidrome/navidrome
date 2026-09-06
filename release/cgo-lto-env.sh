#!/bin/sh
# Emit shell exports for CGO LTO flags.
#
# Prefer clang + lld on Linux (matches Dockerfile.jbs gobuilder).
# Fat mode uses LLVM full LTO (-flto=full); thin mode keeps PGO training fast.
#
# Usage:
#   eval "$(./release/cgo-lto-env.sh thin)"   # PGO profile collection
#   eval "$(./release/cgo-lto-env.sh fat)"    # final optimized build
#   eval "$(./release/cgo-lto-env.sh off)"    # disable LTO
#
# Environment (optional, fat/thin modes):
#   CGO_RELEASE_CFLAGS   extra -O3 / march flags prepended to CGO_CFLAGS
#   CGO_BASE_CFLAGS      base flags before LTO (default: empty)
#   SQLITE_EXTRA_CFLAGS  amalgamation -D flags (default: release/sqlite-cflags.sh
#                        when that script exists; empty otherwise). Applied to
#                        CGO_CFLAGS only so the sqlite amalgamation is compiled
#                        with the same defines under thin and fat LTO.
set -e

mode="${1:-thin}"
cc="${CC:-clang}"

opt_flags="${CGO_RELEASE_CFLAGS:--O3 -ffunction-sections -fdata-sections -fno-semantic-interposition}"
lto_flags=""
lld_flags=""

case "$mode" in
  thin)
    case "$cc" in
      *clang*)
        lto_flags="-flto=thin"
        lld_flags="-fuse-ld=lld"
        ;;
      *)
        lto_flags="-flto=auto"
        ;;
    esac
    ;;
  fat|full)
    case "$cc" in
      *clang*)
        # Aligned with Dockerfile.jbs LLD_FLAGS (relro/now/build-id=none).
        lto_flags="-flto=full"
        lld_flags="-fuse-ld=lld -Wl,-O2 -Wl,--lto-O3 -Wl,--gc-sections -Wl,--icf=safe -Wl,--as-needed -Wl,-z,relro,-z,now -Wl,--build-id=none"
        ;;
      *)
        lto_flags="-flto -flto-partition=one"
        lld_flags="-Wl,--gc-sections"
        ;;
    esac
    ;;
  off)
  ;;
  *)
    echo "usage: $0 thin|fat|off" >&2
    exit 1
    ;;
esac

base_c="${CGO_BASE_CFLAGS:-}"
base_cxx="${CGO_BASE_CXXFLAGS:-}"
base_ld="${CGO_BASE_LDFLAGS:-}"

# Resolve SQLite amalgamation defines once so thin (PGO train) and fat (final)
# LTO builds compile sqlite with identical feature/default flags.
script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
if [ -z "${SQLITE_EXTRA_CFLAGS+x}" ]; then
  if [ -x "${script_dir}/sqlite-cflags.sh" ]; then
    SQLITE_EXTRA_CFLAGS="$("${script_dir}/sqlite-cflags.sh")"
  else
    SQLITE_EXTRA_CFLAGS=""
  fi
fi

if [ "$mode" != "off" ]; then
  base_c="${opt_flags}${base_c:+ ${base_c}}"
  base_cxx="${opt_flags}${base_cxx:+ ${base_cxx}}"
fi

# SQLITE defines belong on CFLAGS (amalgamation is C), not CXXFLAGS.
# FTS5 BM25 needs libm (log). go-sqlite3 adds -lm only with the sqlite_fts5
# build tag; when SQLITE_EXTRA_CFLAGS enables FTS5 via -D (LTO train/final),
# still link libm so mismatched tags cannot break the link.
if printf '%s' "${SQLITE_EXTRA_CFLAGS}" | grep -q 'DSQLITE_ENABLE_FTS5'; then
  case " ${base_ld} " in
    *" -lm "*) ;;
    *) base_ld="${base_ld:+${base_ld} }-lm" ;;
  esac
fi

printf 'export CGO_CFLAGS="%s%s%s"\n' "${base_c}" "${lto_flags:+ ${lto_flags}}" "${SQLITE_EXTRA_CFLAGS:+ ${SQLITE_EXTRA_CFLAGS}}"
printf 'export CGO_CXXFLAGS="%s%s"\n' "${base_cxx}" "${lto_flags:+ ${lto_flags}}"
printf 'export CGO_LDFLAGS="%s%s%s"\n' "${base_ld}" "${lto_flags:+ ${lto_flags}}" "${lld_flags:+ ${lld_flags}}"
