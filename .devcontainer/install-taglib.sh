#!/bin/bash
set -euo pipefail

# 1. Detect Architecture
ARCH=$(dpkg --print-architecture)
TAGLIB_VERSION="v2.1.1-1"

case $ARCH in
    "amd64")
        DOWNLOAD_ARCH="linux-amd64"
        ;;
    "arm64")
        DOWNLOAD_ARCH="linux-arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

URL="https://github.com/navidrome/cross-taglib/releases/download/${TAGLIB_VERSION}/taglib-${DOWNLOAD_ARCH}.tar.gz"

echo "Downloading TagLib for ${ARCH} from ${URL}"

wget "$URL" -O /tmp/cross-taglib.tar.gz
sudo tar -xzf /tmp/cross-taglib.tar.gz -C /usr --strip-components=1
sudo mv /usr/include/taglib/* /usr/include/
sudo rmdir /usr/include/taglib
sudo rm /tmp/cross-taglib.tar.gz /usr/provenance.json

echo "TagLib installation complete"