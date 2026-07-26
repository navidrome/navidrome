#!/bin/sh
# Fail the build if a 32-bit ARM/x86 binary links ebitengine/purego, which would
# SIGSEGV at startup on those arches (issue #5738).
#
# Independent safety net for build-tags.sh: it inspects the actual build metadata
# recorded in the binary (survives stripping) instead of trusting the requested
# tags, so it still fires if the tag decision is wrong or gen2brain/webp changes
# its build-tag semantics. Runs in the Dockerfile, where xx-info and go are present.
#
# Usage: verify-binary.sh <binary> [<binary>...]
set -e

# Prefer xx-info (the cross-build target arch); fall back to `go env GOARCH` so the
# check is still correct when run outside the xx environment.
arch=$(xx-info arch 2>/dev/null || go env GOARCH)

case "${arch}" in
    arm | 386) ;;
    *) exit 0 ;; # 64-bit arches legitimately link purego for native libwebp
esac

for bin in "$@"; do
    # Fail loudly if the expected binary is missing (e.g. an unmatched glob), rather
    # than letting `go version -m` fail inside the pipeline and silently pass.
    if [ ! -f "${bin}" ]; then
        echo "ERROR: expected binary '${bin}' not found; purego verification did not run."
        exit 1
    fi
    if go version -m "${bin}" | grep -q "ebitengine/purego"; then
        echo "ERROR: 32-bit binary '${bin}' links ebitengine/purego; it will SIGSEGV at startup (issue #5738)."
        echo "       Ensure the 'nodynamic' build tag is applied (see release/build-tags.sh)."
        exit 1
    fi
done
