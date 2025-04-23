# Fake Multi Agent Plugin

This directory contains a test plugin for the Navidrome plugin system, implementing both the ArtistMetadataService and AlbumMetadataService interfaces.

## Requirements

- Go 1.24 or newer (with WASI support)
- The Navidrome repository (with generated plugin API code in `plugins/api`)

## How to Compile

To build the WASM plugin, run the following command from the project root:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugins/testdata/fake_multi_agent/plugin.wasm ./plugins/testdata/fake_multi_agent
```

## Behavior

Implements both services. Example responses:

- **GetArtistMBID**: returns `{ Mbid: "multi-artist-mbid" }` if `Name` is not empty
- **GetAlbumInfo**: returns `{ Info: { Name: <name>, Mbid: "multi-album-mbid", Description: "Multi agent album description", Url: "https://multi.example.com/album" } }` if `Name` and `Artist` are not empty

Other methods return simple static or empty responses.

## Usage

This plugin can be loaded by the Navidrome host for integration and end-to-end tests of the plugin system.
