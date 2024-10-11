FROM --platform=$BUILDPLATFORM crazymax/osxcross:11.3-debian AS osxcross
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.5.0 AS xx

#####################################################
### Get TagLib
FROM --platform=$BUILDPLATFORM alpine AS taglib-build
ARG TARGETPLATFORM
RUN --mount=type=bind,source=docker,target=/tmp \
    PLATFORM=$(echo ${TARGETPLATFORM} | tr '/' '_') && \
    echo $PLATFORM && cp -R /tmp/taglib/${PLATFORM} /taglib

#####################################################
### Build Navidrome
FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS base
RUN apt-get update && apt-get install -y clang lld git
COPY --from=xx / /
WORKDIR /workspace

FROM --platform=$BUILDPLATFORM base AS build
ENV CGO_ENABLED=1
ENV PKG_CONFIG_PATH=/taglib/lib/pkgconfig

# Install build dependencies for the target platform
ARG TARGETPLATFORM
RUN xx-apt install -y binutils gcc g++ libc6-dev zlib1g-dev
RUN xx-verify --setup
RUN xx-go --wrap

RUN --mount=type=bind,source=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

RUN --mount=type=bind,source=. \
    --mount=from=osxcross,target=/osxcross,src=/osxcross,ro \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=from=taglib-build,target=/taglib,src=/taglib,ro \
    mkdir -p /xx-sdk && \
    ln -s /osxcross/SDK/MacOSX11* /xx-sdk/MacOSX11.1.sdk && \
    LD_EXTRA="$([ "$(xx-info os)" = "darwin" ] || echo "-extldflags '-static -latomic'")" && \
    cat $(go env GOENV) && echo $LD_EXTRA && \
    go build -ldflags="${LD_EXTRA} -w -s" -o /out/navidrome .

# Verify if the binary was built for the correct platform and it is statically linked
RUN xx-verify --static /out/navidrome

FROM scratch AS binary
COPY --from=build /out /

#####################################################
### Build Final Image
FROM --platform=$BUILDPLATFORM alpine:3.20
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

