// Test plugin that only implements some metadata methods.
// Used to test the "not implemented" code path (-2 return code).
// Build with: tinygo build -o ../partial-metadata-agent.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
)

func init() {
	metadata.Register(&partialMetadataAgent{})
}

// partialMetadataAgent only implements GetArtistBiography.
// All other methods will return NotImplementedCode (-2).
type partialMetadataAgent struct{}

// GetArtistBiography is the only method we implement.
func (t *partialMetadataAgent) GetArtistBiography(input metadata.ArtistRequest) (*metadata.ArtistBiographyResponse, error) {
	return &metadata.ArtistBiographyResponse{Biography: "Partial agent biography for " + input.Name}, nil
}

func main() {}
