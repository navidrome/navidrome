FROM golang:alpine AS build

RUN apk --update add \
    g++ \
    gcc \
    git \
    musl-dev \
    npm \
    pkgconfig \
    taglib-dev \
    zlib-static

RUN mkdir /build

COPY . /build

WORKDIR /build

RUN (cd ./ui && npm ci && npm run build)

RUN go build

FROM alpine:latest

RUN apk --update add \
    taglib-dev

COPY --from=build /build/navidrome /usr/bin/navidrome

ENTRYPOINT ["navidrome"]
