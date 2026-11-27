#!/bin/sh
# Print the Go build tags for the xx-cc target platform (used by the Dockerfile).
#
# gen2brain/webp's native libwebp backend links ebitengine/purego, whose reverse
# callbacks are unsupported on 32-bit ARM and x86 and SIGSEGV at package-init time,
# taking the whole process down at startup (issues #5597 / #5606 / #5738). Force the
# WASM-only path there with the "nodynamic" tag; 64-bit arches keep native libwebp.
#
# This is the single source of truth for the tag decision: both Dockerfile build
# stages (Docker-image and standalone downloads) call it so they cannot drift apart.
set -e

# Prefer xx-info (the cross-build target arch); fall back to `go env GOARCH` so the
# script is still correct when run outside the xx environment. Both report the
# cross-compilation target, unlike `uname -m`, which would report the build host.
arch=$(xx-info arch 2>/dev/null || go env GOARCH)

tags="netgo,sqlite_fts5"
case "${arch}" in
    arm | 386) tags="${tags},nodynamic" ;;
esac
printf '%s' "${tags}"
