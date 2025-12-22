# Minimal Navidrome Plugin Example

This is a minimal example demonstrating how to create a Navidrome plugin using Go and the Extism PDK.

## Building

1. Install [TinyGo](https://tinygo.org/getting-started/install/)
2. Build the plugin:
   ```bash
   go mod tidy
   tinygo build -o minimal.wasm -target wasip1 -buildmode=c-shared ./main.go
   ```

## Installing

Copy `minimal.wasm` to your Navidrome plugins folder (default: `<data-folder>/plugins/`).

## Configuration

Enable plugins in your `navidrome.toml`:

```toml
[Plugins]
Enabled = true

# Add the plugin to your agents list
Agents = "lastfm,spotify,minimal"
```

## What This Example Demonstrates

- Exporting the required `nd_manifest` function
- Implementing `nd_get_artist_biography` as a MetadataAgent capability
- Basic JSON input/output handling with the Extism PDK

## Extending the Example

To add more capabilities, implement additional exported functions:

- `nd_get_artist_mbid` - Get MusicBrainz ID for an artist
- `nd_get_artist_url` - Get external URL for an artist
- `nd_get_similar_artists` - Get similar artists
- `nd_get_artist_images` - Get artist images
- `nd_get_artist_top_songs` - Get top songs for an artist
- `nd_get_album_info` - Get album information
- `nd_get_album_images` - Get album images

See the full documentation in `/plugins/README.md` for input/output formats.
