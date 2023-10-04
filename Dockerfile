#####################################################
### Build UI bundles
FROM node:18-alpine AS jsbuilder
WORKDIR /src
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ .
RUN npm run build


#####################################################
### Build executable
FROM golang:1.21-alpine AS gobuilder

# Download build tools
RUN mkdir -p /src/ui/build
RUN apk add -U --no-cache build-base git
RUN go install github.com/go-bindata/go-bindata/go-bindata@latest

# Download project dependencies
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN go mod tidy
RUN apk update && apk add curl git pkgconfig && curl https://glide.sh/get | sh
RUN apk add --update taglib-dev gcc taglib
RUN apk add gcompat
RUN apk add --update alpine-sdk
RUN apk add build-base zlib-dev

RUN mkdir -p .git/hooks
RUN cd .git/hooks && ln -sf ../../git/* .
# Copy source, test it
COPY . .
COPY --from=jsbuilder /src/build/* /src/ui/build/
COPY --from=jsbuilder /src/build/static/css/* /src/ui/build/static/css/
COPY --from=jsbuilder /src/build/static/js/* /src/ui/build/static/js/
RUN make build
#RUN CGO_ENABLED=1 GOOS=linux go build
#RUN CGO_ENABLED=1 GOOS=linux go test ./...

# Copy UI bundle, build executable
#COPY --from=jsbuilder /src/build/* /src/ui/build/
#COPY --from=jsbuilder /src/build/static/css/* /src/ui/build/static/css/
#COPY --from=jsbuilder /src/build/static/js/* /src/ui/build/static/js/
#RUN rm -rf /src/build/css /src/build/js
#RUN go-bindata -fs -prefix ui/build -tags embed -nocompress -pkg assets -o assets/embedded_gen.go ui/build/... && \
#    go build -ldflags="-X ./consts -X ./consts" -tags=embed

#####################################################
### Build Final Image
FROM alpine as release
LABEL maintainer="deluan@navidrome.org"

COPY --from=gobuilder /src/navidrome /app/
RUN apk add --update taglib-dev gcc taglib

# Install ffmpeg and output build config
RUN apk add --no-cache ffmpeg
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
HEALTHCHECK CMD wget -O- http://localhost:${ND_PORT}/ping || exit 1
WORKDIR /app

ENTRYPOINT ["/app/navidrome"]
