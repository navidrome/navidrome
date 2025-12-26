// Test Artwork plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-artwork.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"encoding/json"
	"strings"

	pdk "github.com/extism/go-pdk"
)

// Manifest types
type Manifest struct {
	Name        string       `json:"name"`
	Author      string       `json:"author"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

type Permissions struct {
	Artwork *ArtworkPermission `json:"artwork,omitempty"`
}

type ArtworkPermission struct {
	Reason string `json:"reason,omitempty"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:        "Test Artwork",
		Author:      "Navidrome Test",
		Version:     "1.0.0",
		Description: "A test artwork plugin for integration testing",
		Permissions: &Permissions{
			Artwork: &ArtworkPermission{
				Reason: "For testing artwork URL generation",
			},
		},
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	pdk.Output(out)
	return 0
}

// TestInput is the input for nd_test_artwork callback.
type TestInput struct {
	ArtworkType string `json:"artwork_type"` // "artist", "album", "track", "playlist"
	ID          string `json:"id"`
	Size        int32  `json:"size"`
}

// TestOutput is the output from nd_test_artwork callback.
type TestOutput struct {
	URL   string  `json:"url,omitempty"`
	Error *string `json:"error,omitempty"`
}

// nd_test_artwork is the test callback that tests the artwork host functions.
//
//go:wasmexport nd_test_artwork
func ndTestArtwork() int32 {
	var input TestInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestOutput{Error: &errStr})
		return 0
	}

	var url string
	var err error

	switch strings.ToLower(input.ArtworkType) {
	case "artist":
		resp, e := ArtworkGetArtistUrl(input.ID, input.Size)
		if e != nil {
			err = e
		} else {
			url = resp.Url
		}
	case "album":
		resp, e := ArtworkGetAlbumUrl(input.ID, input.Size)
		if e != nil {
			err = e
		} else {
			url = resp.Url
		}
	case "track":
		resp, e := ArtworkGetTrackUrl(input.ID, input.Size)
		if e != nil {
			err = e
		} else {
			url = resp.Url
		}
	case "playlist":
		resp, e := ArtworkGetPlaylistUrl(input.ID, input.Size)
		if e != nil {
			err = e
		} else {
			url = resp.Url
		}
	default:
		errStr := "unknown artwork type: " + input.ArtworkType
		pdk.OutputJSON(TestOutput{Error: &errStr})
		return 0
	}

	if err != nil {
		errStr := err.Error()
		pdk.OutputJSON(TestOutput{Error: &errStr})
		return 0
	}

	pdk.OutputJSON(TestOutput{URL: url})
	return 0
}

func main() {}
