FROM --platform=$BUILDPLATFORM ghcr.io/crazy-max/osxcross:14.5-debian AS osxcross

########################################################################################################################
### Build xx (original image: tonistiigi/xx)
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/alpine:3.20 AS xx-build

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
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/node:lts-alpine AS ui
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
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:1.25-alpine AS build-alpine
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
    # -latomic is required on 32-bit arm (arm/v6, arm/v7) so SQLite's 64-bit atomics resolve.
    go build -tags=netgo,sqlite_fts5 -ldflags="-w -s \
        -linkmode=external -extldflags '-latomic' \
        -X github.com/navidrome/navidrome/consts.gitSha=${GIT_SHA} \
        -X github.com/navidrome/navidrome/consts.gitTag=${GIT_TAG}" \
        -o /out/navidrome .
    # Fail the build if the binary is accidentally statically linked: dlopen (and
    # therefore native libwebp detection) only works with a dynamic interpreter.
    file /out/navidrome | grep -q "dynamically linked" || { echo "ERROR: /out/navidrome is not dynamically linked"; file /out/navidrome; exit 1; }
EOT

########################################################################################################################
### Build Navidrome binary for standalone distribution (static glibc, cross-compiled)
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:1.25-trixie AS base
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

    # Setup CGO cross-compilation environment
    xx-go --wrap
    export CGO_ENABLED=1
    cat $(go env GOENV)

    # Only Darwin (macOS) requires clang (default), Windows requires gcc, everything else can use any compiler.
    # So let's use gcc for everything except Darwin.
    if [ "$(xx-info os)" != "darwin" ]; then
        export CC=$(xx-info)-gcc
        export CXX=$(xx-info)-g++
        export LD_EXTRA="-extldflags '-static -latomic'"
    fi
    if [ "$(xx-info os)" = "windows" ]; then
        export EXT=".exe"
    fi

    go build -tags=netgo,sqlite_fts5 -ldflags="${LD_EXTRA} -w -s \
        -X github.com/navidrome/navidrome/consts.gitSha=${GIT_SHA} \
        -X github.com/navidrome/navidrome/consts.gitTag=${GIT_TAG}" \
        -o /out/navidrome${EXT} .
EOT

# Verify if the binary was built for the correct platform and it is statically linked
RUN xx-verify --static /out/navidrome*

FROM scratch AS binary
COPY --from=build /out /

########################################################################################################################
### Build Final Image
FROM public.ecr.aws/docker/library/alpine:3.20 AS final
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
ENV ND_ENABLEWEBPENCODING=true
RUN touch /.nddockerenv

EXPOSE ${ND_PORT}
WORKDIR /app

ENTRYPOINT ["/app/navidrome"]

