# Cover Art Archive AlbumMetadataService Plugin

This plugin provides album cover images for Navidrome by querying the [Cover Art Archive](https://coverartarchive.org/) API using the MusicBrainz Release Group MBID.

## Features

- Implements only the `GetAlbumImages` method of the AlbumMetadataService plugin interface.
- Returns front cover images for a given release-group MBID.
- Returns `not found` if no MBID is provided or no images are found.

## Requirements

- Go 1.24 or newer (with WASI support)
- The Navidrome repository (with generated plugin API code in `plugins/api`)

## How to Compile

To build the WASM plugin, run the following command from the project root:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugins/testdata/coverartarchive/plugin.wasm ./plugins/testdata/coverartarchive
```

This will produce `plugin.wasm` in this directory.

## Usage

- The plugin can be loaded by Navidrome for integration and end-to-end tests of the plugin system.
- It is intended for testing and development purposes only.

## API Reference

- [Cover Art Archive API](https://musicbrainz.org/doc/Cover_Art_Archive/API)
- This plugin uses the endpoint: `https://coverartarchive.org/release-group/{mbid}`
