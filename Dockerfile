# syntax=docker/dockerfile:1.25

FROM --platform=$BUILDPLATFORM ghcr.io/crazy-max/osxcross:14.5-debian AS osxcross

########################################################################################################################
### Build xx (original image: tonistiigi/xx)
FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/alpine:3.24.1 AS xx-build

# v1.9.0
ENV XX_VERSION=a5592eab7a57895e8d385394ff12241bc65ecd50

RUN apk add -U --no-cache git
RUN git clone https://github.com/tonistiigi/xx && \
    cd xx && \
    git checkout ${XX_VERSION} && \
    mkdir -p /out && \
    cp src/xx-* /out/

RUN cd /out && \
    ln -s xx-cc /out/xx-clang && \
    ln -s xx-cc /out/xx-clang++ && \
    ln -s xx-cc /out/xx-c++ && \
    ln -s xx-apt /out/xx-apt-get

# xx mimics the original tonistiigi/xx image
FROM scratch AS xx
COPY --from=xx-build /out/ /usr/bin/

########################################################################################################################
### Build Navidrome UI
FROM --platform=$BUILDPLATFORM oven/bun:canary-alpine@sha256:6cfe3d5c65feda498b5a2018697338d7a7656b959d8f3d03b27e85bc66b4574e AS ui
WORKDIR /app

# Install Bun dependencies
COPY ui/package.json ui/bun.lock ./
COPY ui/bin/ ./bin/
RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile

# Build bundle
COPY ui/ ./
RUN bun run build

FROM scratch AS ui-bundle
COPY --from=ui /app/build /build

########################################################################################################################
### Build Navidrome binary for Docker image (dynamic musl, enables native libwebp via dlopen)
FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/golang:1.27.1-alpine AS build-alpine
COPY --from=xx / /

ARG TARGETPLATFORM

RUN apk add --no-cache clang lld file git
RUN xx-apk add --no-cache gcc musl-dev zlib-dev
RUN xx-verify --setup

WORKDIR /workspace

RUN --mount=type=bind,source=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

ARG GIT_SHA
ARG GIT_TAG
ARG GO_PGO_ENABLED=true
ARG GO_PGO_BENCHTIME=3s

RUN --mount=type=bind,source=. \
    --mount=from=ui,source=/build,target=./ui/build,ro \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -e
    xx-go --wrap
    export CGO_ENABLED=1
    BUILD_TAGS=$(./release/build-tags.sh)
    if [ "${GO_PGO_ENABLED}" = "true" ]; then
      eval "$(./release/cgo-lto-env.sh thin)"
      PGO_BUILD_TAGS="${BUILD_TAGS}" \
        ND_GRPCWORKERINTESTS=1 \
        GO_PGO_BENCHTIME="${GO_PGO_BENCHTIME}" \
        PGO_OUTPUT=/tmp/default.pgo \
        ./release/pgo-train.sh
    fi
    eval "$(./release/cgo-lto-env.sh fat)"
    PGO_FLAG="-pgo=off"
    if [ "${GO_PGO_ENABLED}" = "true" ] && [ -s /tmp/default.pgo ]; then
      PGO_FLAG="-pgo=/tmp/default.pgo"
    fi
    # -latomic is required on 32-bit arm (arm/v6, arm/v7) so SQLite's 64-bit atomics resolve.
    go build -tags="${BUILD_TAGS}" ${PGO_FLAG} -trimpath -buildvcs=false -ldflags="-w -s \
        -linkmode=external -extldflags '-latomic' \
        -X github.com/navidrome/navidrome/consts.gitSha=${GIT_SHA} \
        -X github.com/navidrome/navidrome/consts.gitTag=${GIT_TAG}" \
        -o /out/navidrome .
    # Fail the build if native libwebp (purego) leaked into a 32-bit binary (issue #5738).
    ./release/verify-binary.sh /out/navidrome
    # Fail the build if the binary is accidentally statically linked: dlopen (and
    # therefore native libwebp detection) only works with a dynamic interpreter.
    file /out/navidrome | grep -q "dynamically linked" || { echo "ERROR: /out/navidrome is not dynamically linked"; file /out/navidrome; exit 1; }
EOT

########################################################################################################################
### Build Navidrome binary for standalone distribution (static glibc, cross-compiled)
FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/golang:1.27.1-trixie AS base
RUN apt-get update && apt-get install -y clang lld
COPY --from=xx / /
WORKDIR /workspace

FROM --platform=$BUILDPLATFORM base AS build

# Install build dependencies for the target platform
ARG TARGETPLATFORM

RUN xx-apt install -y binutils gcc g++ libc6-dev zlib1g-dev
RUN xx-verify --setup

RUN --mount=type=bind,source=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

ARG GIT_SHA
ARG GIT_TAG
ARG GO_PGO_ENABLED=true
ARG GO_PGO_BENCHTIME=3s

RUN --mount=type=bind,source=. \
    --mount=from=ui,source=/build,target=./ui/build,ro \
    --mount=from=osxcross,src=/osxcross/SDK,target=/xx-sdk,ro \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -e

    # Setup CGO cross-compilation environment
    xx-go --wrap
    export CGO_ENABLED=1
    cat "$(go env GOENV)" 2>/dev/null || true

    # Only Darwin (macOS) requires clang (default), Windows requires gcc, everything else can use any compiler.
    # So let's use gcc for everything except Darwin.
    if [ "$(xx-info os)" != "darwin" ]; then
        export CC=$(xx-info)-gcc
        export CXX=$(xx-info)-g++
        export LD_EXTRA="-extldflags '-static -latomic'"
    fi
    # GNU ld corrupts the R_ARM_IRELATIVE addends of libatomic's ifunc resolvers
    # (wrong address, Thumb bit lost) once .text outgrows the 16MB Thumb branch
    # range, making static arm binaries jump to garbage inside glibc's ifunc
    # resolution and crash before main() (issue #5738). Link 32-bit arm with LLD,
    # which emits correct addends.
    if [ "$(xx-info arch)" = "arm" ]; then
        export LD_EXTRA="-extldflags '-static -latomic -fuse-ld=lld'"
    fi
    if [ "$(xx-info os)" = "windows" ]; then
        export EXT=".exe"
    fi

    BUILD_TAGS=$(./release/build-tags.sh)
    if [ "${GO_PGO_ENABLED}" = "true" ]; then
      eval "$(./release/cgo-lto-env.sh thin)"
      PGO_BUILD_TAGS="${BUILD_TAGS}" \
        ND_GRPCWORKERINTESTS=1 \
        GO_PGO_BENCHTIME="${GO_PGO_BENCHTIME}" \
        PGO_OUTPUT=/tmp/default.pgo \
        ./release/pgo-train.sh
    fi

    eval "$(./release/cgo-lto-env.sh fat)"
    PGO_FLAG="-pgo=off"
    if [ "${GO_PGO_ENABLED}" = "true" ] && [ -s /tmp/default.pgo ]; then
      PGO_FLAG="-pgo=/tmp/default.pgo"
    fi

    go build -tags="${BUILD_TAGS}" ${PGO_FLAG} -trimpath -buildvcs=false -ldflags="${LD_EXTRA} -w -s \
        -X github.com/navidrome/navidrome/consts.gitSha=${GIT_SHA} \
        -X github.com/navidrome/navidrome/consts.gitTag=${GIT_TAG}" \
        -o /out/navidrome${EXT} .
    # Fail the build if native libwebp (purego) leaked into a 32-bit binary (issue #5738).
    ./release/verify-binary.sh /out/navidrome*
EOT

# Verify if the binary was built for the correct platform and it is statically linked
RUN xx-verify --static /out/navidrome*

FROM scratch AS binary
COPY --from=build /out /

########################################################################################################################
### Build Final Image
FROM mirror.gcr.io/library/alpine:3.24.1 AS final
LABEL maintainer="deluan@navidrome.org"
LABEL org.opencontainers.image.source="https://github.com/navidrome/navidrome"

# Install runtime dependencies
# - libwebp + symlinks: enables native WebP encoding via purego/dlopen
RUN apk add -U --no-cache ffmpeg mpv sqlite libwebp libwebpdemux libwebpmux && \
    for lib in libwebp libwebpdemux libwebpmux; do \
        target=$(ls /usr/lib/$lib.so.* 2>/dev/null | head -1) && \
        [ -n "$target" ] && ln -sf "$target" /usr/lib/$lib.so; \
    done

# Copy navidrome binary (musl build for Docker, enables native libwebp)
COPY --from=build-alpine /out/navidrome /app/

VOLUME ["/data", "/music"]
ENV ND_MUSICFOLDER=/music
ENV ND_DATAFOLDER=/data
ENV ND_CONFIGFILE=/data/navidrome.toml
ENV ND_PORT=4533
# Plaintext H2C gRPC listener for private overlays (WireGuard): gRPC only,
# no TLS. Overridden per deploy, e.g. ND_PUBLICGRPCADDRESS.
ENV ND_PUBLICGRPCPORT=50051
RUN touch /.nddockerenv

EXPOSE ${ND_PORT} ${ND_PUBLICGRPCPORT}
WORKDIR /app
ENV PATH="/app:${PATH}"

ENTRYPOINT ["/app/navidrome"]
