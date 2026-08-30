FROM --platform=$BUILDPLATFORM ghcr.io/crazy-max/osxcross:14.5-debian AS osxcross

########################################################################################################################
### Build xx (original image: tonistiigi/xx)
FROM --platform=$BUILDPLATFORM alpine:3.24 AS xx-build

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
FROM --platform=$BUILDPLATFORM node:lts-alpine AS ui
WORKDIR /app

# Install node dependencies
COPY ui/package.json ui/package-lock.json ./
COPY ui/bin/ ./bin/
RUN npm ci

# Build bundle
COPY ui/ ./
RUN npm run build -- --outDir=/build

FROM scratch AS ui-bundle
COPY --from=ui /build /build

########################################################################################################################
### Build Navidrome binary for Docker image (dynamic musl, enables native libwebp via dlopen)
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build-alpine
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

RUN --mount=type=bind,source=. \
    --mount=from=ui,source=/build,target=./ui/build,ro \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -e
    xx-go --wrap
    export CGO_ENABLED=1
    BUILD_TAGS=$(./release/build-tags.sh)
    # -latomic is required on 32-bit arm (arm/v6, arm/v7) so SQLite's 64-bit atomics resolve.
    go build -tags="${BUILD_TAGS}" -ldflags="-w -s \
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
FROM --platform=$BUILDPLATFORM golang:1.27-trixie AS base
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
    go build -tags="${BUILD_TAGS}" -ldflags="${LD_EXTRA} -w -s \
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
### Build no-op stubs for mpv's video-output libraries
# mpv links libEGL/libgbm for video output only; Navidrome drives it headless, for audio.
# Real mesa pulls in LLVM + gallium (+218MB uncompressed), so ship stubs it never calls.
FROM --platform=$BUILDPLATFORM alpine:3.24 AS mpv-stubs
COPY --from=xx / /
RUN apk add --no-cache clang lld binutils mesa-egl mesa-gbm
ARG TARGETPLATFORM
RUN xx-apk add --no-cache musl-dev
RUN <<EOT
    set -e
    mkdir -p /out
    for so in libEGL.so.1 libgbm.so.1; do
        readelf -sW /usr/lib/$so \
            | awk '$5 == "GLOBAL" && $7 != "UND" { print $8 }' \
            | sed 's/@.*//' \
            | grep -vE '^(_init|_fini|_edata|_end|__bss_start|_GLOBAL_OFFSET_TABLE_)$' \
            | sort -u \
            | awk '{ print "void " $1 "(void) {}" }' > /tmp/stub.c
        test -s /tmp/stub.c
        xx-clang -shared -nostdlib -fPIC -Wl,-soname,$so -o /out/$so /tmp/stub.c
        xx-verify /out/$so
    done
EOT

########################################################################################################################
### Build Final Image
FROM alpine:3.24 AS final
LABEL maintainer="deluan@navidrome.org"
LABEL org.opencontainers.image.source="https://github.com/navidrome/navidrome"

# Install runtime dependencies
# - libwebp + symlinks: enables native WebP encoding via purego/dlopen
# The mesa/LLVM stack mpv pulls in for video output is dropped in this same layer,
# otherwise the deleted bytes still ship in the image.
RUN apk add -U --no-cache ffmpeg mpv sqlite libwebp libwebpdemux libwebpmux && \
    for lib in libwebp libwebpdemux libwebpmux; do \
        target=$(ls /usr/lib/$lib.so.* 2>/dev/null | head -1) && \
        [ -n "$target" ] && ln -sf "$target" /usr/lib/$lib.so; \
    done && \
    rm -rf /usr/lib/gallium-pipe /usr/lib/dri \
        /usr/lib/libEGL.so* /usr/lib/libgbm.so* /usr/lib/libgallium*.so /usr/lib/libLLVM.so* \
        /usr/lib/libGL.so* /usr/lib/libGLESv2.so* /usr/lib/libglapi.so*

COPY --from=mpv-stubs /out/ /usr/lib/
RUN mpv --no-video --ao=null --version > /dev/null

# Copy navidrome binary (musl build for Docker, enables native libwebp)
COPY --from=build-alpine /out/navidrome /app/

VOLUME ["/data", "/music"]
ENV ND_MUSICFOLDER=/music
ENV ND_DATAFOLDER=/data
ENV ND_CONFIGFILE=/data/navidrome.toml
ENV ND_PORT=4533
RUN touch /.nddockerenv

EXPOSE ${ND_PORT}
WORKDIR /app
ENV PATH="/app:${PATH}"

ENTRYPOINT ["/app/navidrome"]

