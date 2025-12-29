// Minimal example Navidrome plugin demonstrating the MetadataAgent capability.
//
// Build with:
//
//	tinygo build -o minimal.wasm -target wasip1 -buildmode=c-shared ./main.go
//
// Install by copying minimal.ndp to your Navidrome plugins folder.
package main

import (
	"github.com/extism/go-pdk"
)

type ArtistInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

type BiographyOutput struct {
	Biography string `json:"biography"`
}

//go:wasmexport nd_get_artist_biography
func ndGetArtistBiography() int32 {
	var input ArtistInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	output := BiographyOutput{
		Biography: "This is a placeholder biography for " + input.Name + ".",
	}

	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

func main() {}
