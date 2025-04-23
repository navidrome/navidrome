# Minimal Fake Album Test Agent Plugin

This directory contains a minimal test plugin for the Navidrome plugin system, used for testing the AlbumMetadataService plugin infrastructure.

## Requirements

- Go 1.24 or newer (with WASI support)
- The Navidrome repository (with generated plugin API code in `plugins/api`)

## How to Compile

To build the WASM plugin, run the following command from the project root:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugins/testdata/fake_album_agent/plugin.wasm ./plugins/testdata/fake_album_agent
```

This will produce `plugin.wasm` in this directory.

## Behavior

This plugin implements all methods required by the generated `AlbumMetadataService` interface. It is implemented in [`plugin.go`](plugin.go).

For requests where the relevant name and artist fields are not empty, it returns the following test data:

- **GetAlbumInfo**: returns `{ Info: { Name: <name>, MBid: "album-mbid-123", Description: "This is a test album description", Url: "https://example.com/album" } }` if `Name` and `Artist` are not empty
- **GetAlbumImages**: returns `{ Images: [ { Url: "https://example.com/album1.jpg", Size: 300 }, { Url: "https://example.com/album2.jpg", Size: 400 } ] }` if `Name` and `Artist` are not empty

If the required fields are empty, all methods return a `not found` error.

## Notes

- The plugin is intended for testing and development purposes only.
- Make sure the generated files in `plugins/api` are up to date before building the plugin. You can regenerate them with:

```sh
make plugins-gen
```

## Usage

This plugin can be loaded by the Navidrome host for integration and end-to-end tests of the plugin system.
