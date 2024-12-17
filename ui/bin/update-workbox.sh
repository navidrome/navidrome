#!/usr/bin/env sh

set -e

export WORKBOX_DIR=public/3rdparty/workbox

rm -rf ${WORKBOX_DIR}
workbox copyLibraries build/3rdparty/

mkdir -p ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-sw.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-core.prod.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-strategies.prod.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-routing.prod.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-navigation-preload.prod.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-precaching.prod.js ${WORKBOX_DIR}
rm -rf build/3rdparty/workbox-*
