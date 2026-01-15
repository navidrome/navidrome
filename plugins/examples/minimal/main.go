// Minimal example Navidrome plugin demonstrating the MetadataAgent capability.
//
// Build with:
//
//	tinygo build -o minimal.wasm -target wasip1 -buildmode=c-shared .
//
// Install by copying minimal.ndp to your Navidrome plugins folder.
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
)

// minimalPlugin implements the metadata provider interfaces.
type minimalPlugin struct{}

// init registers the plugin implementation
func init() {
	metadata.Register(&minimalPlugin{})
}

var _ metadata.ArtistBiographyProvider = (*minimalPlugin)(nil)

// GetArtistBiography returns a placeholder biography for the artist.
func (p *minimalPlugin) GetArtistBiography(input metadata.ArtistRequest) (*metadata.ArtistBiographyResponse, error) {
	return &metadata.ArtistBiographyResponse{
		Biography: "This is a placeholder biography for " + input.Name + ".",
	}, nil
}

func main() {}
