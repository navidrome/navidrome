#!/bin/sh

#Exit if GIT_TAG is not set
if [ -z "$GIT_TAG" ]; then
  echo "GIT_TAG is not set, exiting..."
  exit 1
fi

set -e

WORKSPACE=$1
ARCH=$2
NAVIDROME_BUILD_VERSION=$(echo "$GIT_TAG" | sed -e 's/^v//' -e 's/-SNAPSHOT/.1/')

echo "Building MSI package for $ARCH, version $NAVIDROME_BUILD_VERSION"

MSI_OUTPUT_DIR=$WORKSPACE/binaries/msi
mkdir -p "$MSI_OUTPUT_DIR"
BINARY_DIR=$WORKSPACE/binaries/windows_${ARCH}

if [ "$ARCH" = "386" ]; then
  PLATFORM="x86"
else
  PLATFORM="x64"
fi

BINARY=$BINARY_DIR/navidrome.exe
if [ ! -f "$BINARY" ]; then
  echo
  echo "$BINARY not found!"
  echo "Build it with 'make single GOOS=windows GOARCH=${ARCH}'"
  exit 1
fi

wget -O "$WORKSPACE/ffmpeg.zip" "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
unzip -d "$WORKSPACE/extracted_ffmpeg" "$WORKSPACE/ffmpeg.zip" "*/ffmpeg.exe"
cp "$WORKSPACE/extracted_ffmpeg/ffmpeg-*-essentials_build/bin/ffmpeg.exe" "$MSI_OUTPUT_DIR"

cp "$WORKSPACE"/LICENSE "$WORKSPACE"/README.md "$MSI_OUTPUT_DIR"
cp "$BINARY" "$MSI_OUTPUT_DIR"

# workaround for wixl WixVariable not working to override bmp locations
cp "$WORKSPACE"/release/wix/bmp/banner.bmp /usr/share/wixl-*/ext/ui/bitmaps/bannrbmp.bmp
cp "$WORKSPACE"/release/wix/bmp/dialogue.bmp /usr/share/wixl-*/ext/ui/bitmaps/dlgbmp.bmp

cd "$MSI_OUTPUT_DIR"
rm -f "$MSI_OUTPUT_DIR"/navidrome_"${ARCH}".msi
wixl "$WORKSPACE"/release/wix/navidrome.wxs -D Version="$NAVIDROME_BUILD_VERSION" -D Platform=$PLATFORM --arch $PLATFORM \
    --ext ui --output "$MSI_OUTPUT_DIR"/navidrome_"${ARCH}".msi

