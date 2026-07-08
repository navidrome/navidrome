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

case "$(xx-info arch)" in
    arm | 386) ;;
    *) exit 0 ;; # 64-bit arches legitimately link purego for native libwebp
esac

for bin in "$@"; do
    if go version -m "${bin}" | grep -q "ebitengine/purego"; then
        echo "ERROR: 32-bit binary '${bin}' links ebitengine/purego; it will SIGSEGV at startup (issue #5738)."
        echo "       Ensure the 'nodynamic' build tag is applied (see release/build-tags.sh)."
        exit 1
    fi
done
