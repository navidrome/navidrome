// Minimal example Navidrome plugin demonstrating the MetadataAgent capability.
//
// Build with:
//
//	tinygo build -o minimal.wasm -target wasip1 -buildmode=c-shared ./main.go
//
// Install by copying minimal.wasm to your Navidrome plugins folder.
package main

import (
	"encoding/json"

	"github.com/extism/go-pdk"
)

type Manifest struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type ArtistInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

type BiographyOutput struct {
	Biography string `json:"biography"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:        "Minimal Example",
		Author:      "Navidrome",
		Version:     "1.0.0",
		Description: "A minimal example plugin",
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	pdk.Output(out)
	return 0
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
