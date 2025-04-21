# Minimal Test Agent Plugin

This directory contains a minimal test plugin for the Navidrome plugin system, used for testing the plugin infrastructure.

## Requirements

- Go 1.24 or newer (with WASI support)
- The Navidrome repository (with generated plugin API code in `plugins/api`)

## How to Compile

To build the WASM plugin, run the following command from the project root:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugins/testdata/agent/plugin.wasm ./plugins/testdata/agent
```

This will produce `plugin.wasm` in this directory.

## Behavior

This plugin implements all methods required by the generated `ArtistMetadataService` interface. It is implemented in [`plugin.go`](plugin.go).

For requests where the relevant name field is not empty, it returns the following test data:

- **GetArtistMBID**: returns `{ Mbid: "1234567890" }` if `Name` is not empty
- **GetArtistURL**: returns `{ Url: "https://example.com" }` if `Name` is not empty
- **GetArtistBiography**: returns `{ Biography: "This is a test biography" }` if `Name` is not empty
- **GetSimilarArtists**: returns `{ Artists: [ { Name: "Similar Artist 1", Mbid: "mbid1" }, { Name: "Similar Artist 2", Mbid: "mbid2" } ] }` if `Name` is not empty
- **GetArtistImages**: returns `{ Images: [ { Url: "https://example.com/image1.jpg", Size: 100 }, { Url: "https://example.com/image2.jpg", Size: 200 } ] }` if `Name` is not empty
- **GetArtistTopSongs**: returns `{ Songs: [ { Name: "Song 1", Mbid: "mbid1" }, { Name: "Song 2", Mbid: "mbid2" } ] }` if `ArtistName` is not empty

If the required name field is empty, all methods return a `not implemented` error.

## Notes

- The plugin is intended for testing and development purposes only.
- Make sure the generated files in `plugins/api` are up to date before building the plugin. You can regenerate them with:

```sh
make plugins-gen
```

## Usage

This plugin can be loaded by the Navidrome host for integration and end-to-end tests of the plugin system.
