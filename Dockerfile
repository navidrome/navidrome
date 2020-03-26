#####################################################
### Build UI bundles
FROM node:13.10-alpine AS jsbuilder
WORKDIR /src
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ .
RUN npm run build


#####################################################
### Build executable
FROM golang:1.14-alpine AS gobuilder

# Download build tools
RUN mkdir -p /src/ui/build
RUN apk add -U --no-cache build-base git
RUN go get -u github.com/go-bindata/go-bindata/...

# Download and unpack static ffmpeg
ARG FFMPEG_URL=https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
RUN wget -O /tmp/ffmpeg.tar.xz ${FFMPEG_URL}
RUN cd /tmp && tar xJf ffmpeg.tar.xz && rm ffmpeg.tar.xz

# Download project dependencies
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# Copy source, test it
COPY . .
RUN go test ./...

# Copy UI bundle, build executable
COPY --from=jsbuilder /src/build/* /src/ui/build/
COPY --from=jsbuilder /src/build/static/css/* /src/ui/build/static/css/
COPY --from=jsbuilder /src/build/static/js/* /src/ui/build/static/js/
RUN rm -rf /src/build/css /src/build/js
RUN GIT_TAG=$(git describe --tags `git rev-list --tags --max-count=1`) && \
    GIT_TAG=${GIT_TAG#"tags/"} && \
    GIT_SHA=$(git rev-parse --short HEAD) && \
    echo "Building version: ${GIT_TAG} (${GIT_SHA})" && \
    go-bindata -fs -prefix ui/build -tags embed -nocompress -pkg assets -o assets/embedded_gen.go ui/build/... && \
    go build -ldflags="-X github.com/deluan/navidrome/consts.gitSha=${GIT_SHA} -X github.com/deluan/navidrome/consts.gitTag=${GIT_TAG}" -tags=embed

#####################################################
### Build Final Image
FROM alpine as release
MAINTAINER  Deluan Quintao <navidrome@deluan.com>

# Download Tini
ENV TINI_VERSION v0.18.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static /tini
RUN chmod +x /tini

COPY --from=gobuilder /src/navidrome /app/
COPY --from=gobuilder /tmp/ffmpeg*/ffmpeg /usr/bin/

# Check if ffmpeg runs properly
RUN ffmpeg -buildconf

VOLUME ["/data", "/music"]
ENV ND_MUSICFOLDER /music
ENV ND_DATAFOLDER /data
ENV ND_SCANINTERVAL 1m
ENV ND_TRANSCODINGCACHESIZE 100MB
ENV ND_SESSIONTIMEOUT 30m
ENV ND_LOGLEVEL info
ENV ND_PORT 4533

EXPOSE ${ND_PORT}
HEALTHCHECK CMD wget -O- http://localhost:4533/ping || exit 1
WORKDIR /app

ENTRYPOINT ["/tini", "--"]
CMD ["/app/navidrome"]
