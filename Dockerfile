FROM --platform=$BUILDPLATFORM ghcr.io/crazy-max/osxcross:14.5-debian AS osxcross

########################################################################################################################
### Build xx (orignal image: tonistiigi/xx)
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/alpine:3.20 AS xx-build

# v1.5.0
ENV XX_VERSION=b4e4c451c778822e6742bfc9d9a91d7c7d885c8a

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
### Get TagLib
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/alpine:3.20 AS taglib-build
ARG TARGETPLATFORM
ARG CROSS_TAGLIB_VERSION=2.0.2-1
ENV CROSS_TAGLIB_RELEASES_URL=https://github.com/navidrome/cross-taglib/releases/download/v${CROSS_TAGLIB_VERSION}/

RUN <<EOT
    # Store all checksums statically here, to validate the downloaded files.
    # Update this list when setting CROSS_TAGLIB_VERSION
    CROSS_TAGLIB_CHECKSUMS="
0ef8bd9076dea3f26413a4e46765c4ff083e2a6ad0ad046c8a568fe1d7958362  taglib-darwin-amd64.tar.gz
824c795ea4054a172137241eed99f42b073d8e7fb3f3ac497138ce351f1b46df  taglib-darwin-arm64.tar.gz
3a623a2376832ed42f21209d78ef1d7806ffd48de05fc7329adfc09eba8ecbe7  taglib-linux-386.tar.gz
1d7998f44663a615bf0e0ab005c4992128fa83f8111e616cbb1b46ef9ea2ce75  taglib-linux-amd64.tar.gz
10e5fa8057fe1da53be945d2682516dd45b8530dadc9d53b65b2b26676ae5863  taglib-linux-arm-v5.tar.gz
1115832ff6de62d2a0fe46828797210d56f84ec7918d169993710cb4981107c9  taglib-linux-arm-v6.tar.gz
50543937b90dbd45d82afcbebc1363c8b08ed4dba6a209a0f7186f8447ded3c9  taglib-linux-arm-v7.tar.gz
41642093505316f9abf41551f00ba4cf6521f0a84219776a6ad5da240b5d4c98  taglib-linux-arm64.tar.gz
5e5fec3e6277e5073866018e44523a334cac46f3013c2d5923f8fdf20f291cfa  taglib-windows-386.tar.gz
45756d7db17a63d795af5738e7e1b51f4de4f51bb659e5dfd29661a8ca2f95b6  taglib-windows-amd64.tar.gz
"
    PLATFORM=$(echo ${TARGETPLATFORM} | tr '/' '-')
    FILE=taglib-${PLATFORM}.tar.gz
    CHECKSUM=$(echo "${CROSS_TAGLIB_CHECKSUMS}" | grep "${FILE}" | awk '{print $1}')

    DOWNLOAD_URL=${CROSS_TAGLIB_RELEASES_URL}${FILE}
    wget ${DOWNLOAD_URL}
    echo "${CHECKSUM} ${FILE}" | sha256sum -c - || exit 1;

    mkdir /taglib
    tar -xzf ${FILE} -C /taglib
EOT

########################################################################################################################
### Build Navidrome UI
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/node:lts-alpine3.20 AS ui
WORKDIR /app

# Install node dependencies
COPY ui/package.json ui/package-lock.json ./
RUN npm ci

# Build bundle
COPY ui/ ./
RUN npm run build -- --outDir=/build

FROM scratch AS ui-bundle
COPY --from=ui /build /build

########################################################################################################################
### Build Navidrome binary
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:1.23-bookworm AS base
RUN apt-get update && apt-get install -y clang lld
COPY --from=xx / /
WORKDIR /workspace

FROM --platform=$BUILDPLATFORM base AS build

# Install build dependencies for the target platform
ARG TARGETPLATFORM
ARG GIT_SHA
ARG GIT_TAG

RUN xx-apt install -y binutils gcc g++ libc6-dev zlib1g-dev
RUN xx-verify --setup

RUN --mount=type=bind,source=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

RUN --mount=type=bind,source=. \
    --mount=from=ui,source=/build,target=./ui/build,ro \
    --mount=from=osxcross,src=/osxcross/SDK,target=/xx-sdk,ro \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=from=taglib-build,target=/taglib,src=/taglib,ro <<EOT

    # Setup CGO cross-compilation environment
    xx-go --wrap
    export CGO_ENABLED=1
    export PKG_CONFIG_PATH=/taglib/lib/pkgconfig
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

    go build -tags=netgo -ldflags="${LD_EXTRA} -w -s \
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
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/alpine:3.20 AS final
LABEL maintainer="deluan@navidrome.org"

# Install ffmpeg and mpv
RUN apk add -U --no-cache ffmpeg mpv

# Copy navidrome binary
COPY --from=build /out/navidrome /app/

VOLUME ["/data", "/music"]
ENV ND_MUSICFOLDER=/music
ENV ND_DATAFOLDER=/data
ENV ND_PORT=4533
ENV GODEBUG="asyncpreemptoff=1"

EXPOSE ${ND_PORT}
HEALTHCHECK CMD wget -O- http://localhost:${ND_PORT}/ping || exit 1
WORKDIR /app

ENTRYPOINT ["/app/navidrome"]

