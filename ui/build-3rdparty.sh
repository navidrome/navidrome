#!/usr/bin/env sh

workbox copyLibraries build/3rdparty/ && \
mkdir -p build/3rdparty/workbox && \
mv build/3rdparty/workbox-*/workbox-sw.js* build/3rdparty/workbox/ && \
mv build/3rdparty/workbox-*/workbox-core.prod.js* build/3rdparty/workbox/ && \
mv build/3rdparty/workbox-*/workbox-strategies.prod.js* build/3rdparty/workbox/ && \
mv build/3rdparty/workbox-*/workbox-routing.prod.js* build/3rdparty/workbox/ && \
mv build/3rdparty/workbox-*/workbox-navigation-preload.prod.js* build/3rdparty/workbox/ && \
rm -rf build/3rdparty/workbox-*
