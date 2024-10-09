FROM --platform=$BUILDPLATFORM crazymax/osxcross:11.3-debian AS osxcross
FROM --platform=$BUILDPLATFORM tonistiigi/xx AS xx

#####################################################
### Get TagLib
FROM --platform=$BUILDPLATFORM alpine AS taglib-build
ARG TARGETPLATFORM
RUN --mount=type=bind,source=docker,target=/tmp \
    PLATFORM=$(echo ${TARGETPLATFORM} | tr '/' '_') && \
    echo $PLATFORM && cp -R /tmp/taglib/${PLATFORM} /taglib

#####################################################
### Build Navidrome
FROM --platform=$BUILDPLATFORM golang:bookworm AS build
RUN apt-get update && apt-get install -y clang lld git
COPY --from=xx / /
#ENV GO111MODULE=auto
ENV CGO_ENABLED=1
WORKDIR /workspace

COPY --from=taglib-build /taglib /taglib
ENV PKG_CONFIG_PATH=/taglib/lib/pkgconfig

ARG TARGETPLATFORM
RUN xx-apt install -y binutils gcc g++ libc6-dev zlib1g-dev

RUN --mount=type=bind,source=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    xx-go mod download

RUN --mount=type=bind,source=. \
    --mount=from=osxcross,target=/osxcross,src=/osxcross,rw \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    mkdir -p /xx-sdk && \
    ln -s /osxcross/SDK/MacOSX11* /xx-sdk/MacOSX11.1.sdk && \
    STATIC="$([ "$(xx-info os)" = "darwin" ] || echo "-static")" && echo ${STATIC} && \
    xx-go build -ldflags="-extldflags '-lz ${STATIC}' -w -s" -o /out/navidrome .

RUN xx-verify --static /out/navidrome

FROM scratch AS binary
COPY --from=build /out /

#####################################################
### Build Final Image
FROM --platform=$BUILDPLATFORM alpine:3.20
LABEL maintainer="deluan@navidrome.org"

# Install ffmpeg and mpv
RUN apk add -U --no-cache ffmpeg mpv

# Show ffmpeg build info, for troubleshooting purposes
RUN ffmpeg -buildconf

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

