#!/bin/sh
set -e

WORKSPACE=$1
ARCH=$2
NAVIDROME_BUILD_VERSION=$(jq -r '.version' < "$WORKSPACE"/dist/metadata.json | sed -e 's/^v//' -e 's/-SNAPSHOT/.1/' )

echo
echo "Building MSI package for $ARCH, version $NAVIDROME_BUILD_VERSION"

MSI_OUTPUT_DIR=$WORKSPACE/dist/msi
mkdir -p "$MSI_OUTPUT_DIR"
BINARY_DIR=$WORKSPACE/dist/navidrome_windows_${ARCH}_windows_${ARCH}

if [ "$ARCH" = "386" ]; then
  PLATFORM="x86"
else
  PLATFORM="x64"
  BINARY_DIR=${BINARY_DIR}_v1
fi


BINARY=$BINARY_DIR/navidrome.exe
if [ ! -f "$BINARY" ]; then
  echo
  echo "$BINARY not found!"
  echo "Build it with 'make single GOOS=windows GOARCH=${ARCH}'"
  exit 1
fi

cp "$WORKSPACE"/LICENSE $WORKSPACE/README.md "$MSI_OUTPUT_DIR"
cp "$BINARY" "$MSI_OUTPUT_DIR"

# workaround for wixl WixVariable not working to override bmp locations
cp "$WORKSPACE"/wix/bmp/banner.bmp /usr/share/wixl-*/ext/ui/bitmaps/bannrbmp.bmp
cp "$WORKSPACE"/wix/bmp/dialogue.bmp /usr/share/wixl-*/ext/ui/bitmaps/dlgbmp.bmp

cd "$MSI_OUTPUT_DIR"
rm -f "$MSI_OUTPUT_DIR"/*.msi
wixl "$WORKSPACE"/wix/navidrome.wxs -D Version="$NAVIDROME_BUILD_VERSION" -D Platform=$PLATFORM --arch $PLATFORM --ext ui --output "$MSI_OUTPUT_DIR"/navidrome_${ARCH}.msi
du -h "$MSI_OUTPUT_DIR"/*.msi

