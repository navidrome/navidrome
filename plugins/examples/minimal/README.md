# Minimal Navidrome Plugin Example

This is a minimal example demonstrating how to create a Navidrome plugin using Go and the Navidrome PDK.

## Building

1. Install [TinyGo](https://tinygo.org/getting-started/install/)
2. Build the plugin:
   ```bash
   go mod tidy
   tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared .
   zip -j minimal.ndp manifest.json plugin.wasm
   ```

Or using the examples Makefile:
   ```bash
   cd plugins/examples
   make minimal.ndp
   ```

## Installing

Copy `minimal.ndp` to your Navidrome plugins folder (default: `<data-folder>/plugins/`).

## Configuration

Enable plugins in your `navidrome.toml`:

```toml
[Plugins]
Enabled = true

# Add the plugin to your agents list
Agents = "lastfm,spotify,minimal"
```

## What This Example Demonstrates

- Plugin package structure (`.ndp` = zip with `manifest.json` + `plugin.wasm`)
- Using the Navidrome PDK `metadata` subpackage
- Implementing the `ArtistBiographyProvider` interface
- Registration pattern with `metadata.Register()`

## PDK Usage

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/metadata"

type myPlugin struct{}

func init() {
    metadata.Register(&myPlugin{})
}

func (p *myPlugin) GetArtistBiography(input metadata.ArtistRequest) (metadata.ArtistBiographyResponse, error) {
    return metadata.ArtistBiographyResponse{Biography: "..."}, nil
}
```

## Extending the Example

To add more capabilities, implement additional provider interfaces from the `metadata` package:

- `ArtistMBIDProvider` - Get MusicBrainz ID for an artist
- `ArtistURLProvider` - Get external URL for an artist
- `SimilarArtistsProvider` - Get similar artists
- `ArtistImagesProvider` - Get artist images
- `ArtistTopSongsProvider` - Get top songs for an artist
- `AlbumInfoProvider` - Get album information
- `AlbumImagesProvider` - Get album images

See the full documentation in `/plugins/README.md` for input/output formats.
