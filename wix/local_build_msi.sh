#!/bin/sh
set -e

echo -e "\nBuilding MSI package"
WORKSPACE=/workspace
MSI_OUTPUT_DIR=$WORKSPACE/dist/msi

NAVIDROME_BUILD_VERSION=$(jq -r '.version' < $WORKSPACE/dist/metadata.json | sed -e 's/^v//' -e 's/-SNAPSHOT/.1/' )

mkdir -p $MSI_OUTPUT_DIR

BINARY=$WORKSPACE/dist/navidrome_windows_amd64_windows_amd64_v1/navidrome.exe
[ -f $BINARY ] || { echo -e "\nnavidrome.exe not found!\nBuild it with 'make single GOOS=windows GOARCH=amd64'\n"; exit 1; }

cp $WORKSPACE/LICENSE $WORKSPACE/README.md $MSI_OUTPUT_DIR
cp $BINARY $MSI_OUTPUT_DIR

# workaround for wixl WixVariable not working to override bmp locations
cp $WORKSPACE/wix/bmp/banner.bmp /usr/share/wixl-*/ext/ui/bitmaps/bannrbmp.bmp
cp $WORKSPACE/wix/bmp/dialogue.bmp /usr/share/wixl-*/ext/ui/bitmaps/dlgbmp.bmp

cd $MSI_OUTPUT_DIR
rm -f $MSI_OUTPUT_DIR/*.msi
wixl $WORKSPACE/wix/navidrome.wxs -D Version=$NAVIDROME_BUILD_VERSION -D Platform=x64 --arch x64 --ext ui --output $MSI_OUTPUT_DIR/navidrome_amd64.msi
du -h $MSI_OUTPUT_DIR/*.msi

