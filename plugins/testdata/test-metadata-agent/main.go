// Test plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-plugin.wasm -target wasip1 -buildmode=c-shared ./main.go
package main

import (
	"encoding/json"
	"strconv"

	"github.com/extism/go-pdk"
)

// checkConfigError checks if the plugin is configured to return an error.
// If "error" config is set, it returns the error message and exit code.
// If "exitcode" is also set, it uses that value (default: 1).
func checkConfigError() (bool, int32) {
	errMsg, hasErr := pdk.GetConfig("error")
	if !hasErr || errMsg == "" {
		return false, 0
	}
	exitCode := int32(1)
	if code, hasCode := pdk.GetConfig("exitcode"); hasCode {
		if parsed, err := strconv.Atoi(code); err == nil {
			exitCode = int32(parsed)
		}
	}
	pdk.SetErrorString(errMsg)
	return true, exitCode
}

type Manifest struct {
	Name         string       `json:"name"`
	Author       string       `json:"author"`
	Version      string       `json:"version"`
	Description  string       `json:"description"`
	Capabilities []string     `json:"capabilities"`
	Permissions  *Permissions `json:"permissions,omitempty"`
}

type Permissions struct {
	HTTP *HTTPPermission `json:"http,omitempty"`
}

type HTTPPermission struct {
	Reason      string              `json:"reason,omitempty"`
	AllowedURLs map[string][]string `json:"allowedUrls,omitempty"`
}

type ArtistInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

type ArtistInputWithLimit struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	MBID  string `json:"mbid,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type ArtistInputWithCount struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	MBID  string `json:"mbid,omitempty"`
	Count int    `json:"count,omitempty"`
}

type AlbumInput struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
	MBID   string `json:"mbid,omitempty"`
}

type MBIDOutput struct {
	MBID string `json:"mbid"`
}

type URLOutput struct {
	URL string `json:"url"`
}

type BiographyOutput struct {
	Biography string `json:"biography"`
}

type ArtistImage struct {
	URL  string `json:"url"`
	Size int    `json:"size"`
}

type ImagesOutput struct {
	Images []ArtistImage `json:"images"`
}

type SimilarArtist struct {
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

type SimilarArtistsOutput struct {
	Artists []SimilarArtist `json:"artists"`
}

type TopSong struct {
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

type TopSongsOutput struct {
	Songs []TopSong `json:"songs"`
}

type AlbumInfoOutput struct {
	Name        string `json:"name"`
	MBID        string `json:"mbid,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

type AlbumImagesOutput struct {
	Images []ArtistImage `json:"images"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:         "Test Plugin",
		Author:       "Navidrome Test",
		Version:      "1.0.0",
		Description:  "A test plugin for integration testing",
		Capabilities: []string{"MetadataAgent"},
		Permissions: &Permissions{
			HTTP: &HTTPPermission{
				Reason: "Test HTTP access",
				AllowedURLs: map[string][]string{
					"https://test.example.com/*": {"GET"},
				},
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

//go:wasmexport nd_get_artist_mbid
func ndGetArtistMBID() int32 {
	if hasErr, code := checkConfigError(); hasErr {
		return code
	}
	var input ArtistInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}
	output := MBIDOutput{MBID: "test-mbid-" + input.Name}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport nd_get_artist_url
func ndGetArtistURL() int32 {
	if hasErr, code := checkConfigError(); hasErr {
		return code
	}
	var input ArtistInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}
	output := URLOutput{URL: "https://test.example.com/artist/" + input.Name}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport nd_get_artist_biography
func ndGetArtistBiography() int32 {
	if hasErr, code := checkConfigError(); hasErr {
		return code
	}
	var input ArtistInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}
	output := BiographyOutput{Biography: "Biography for " + input.Name}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport nd_get_artist_images
func ndGetArtistImages() int32 {
	if hasErr, code := checkConfigError(); hasErr {
		return code
	}
	var input ArtistInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}
	output := ImagesOutput{
		Images: []ArtistImage{
			{URL: "https://test.example.com/images/" + input.Name + "/large.jpg", Size: 500},
			{URL: "https://test.example.com/images/" + input.Name + "/small.jpg", Size: 100},
		},
	}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport nd_get_similar_artists
func ndGetSimilarArtists() int32 {
	if hasErr, code := checkConfigError(); hasErr {
		return code
	}
	var input ArtistInputWithLimit
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}
	limit := input.Limit
	if limit == 0 {
		limit = 5
	}
	artists := make([]SimilarArtist, 0, limit)
	for i := range limit {
		artists = append(artists, SimilarArtist{
			Name: input.Name + " Similar " + string(rune('A'+i)),
		})
	}
	output := SimilarArtistsOutput{Artists: artists}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport nd_get_artist_top_songs
func ndGetArtistTopSongs() int32 {
	if hasErr, code := checkConfigError(); hasErr {
		return code
	}
	var input ArtistInputWithCount
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}
	count := input.Count
	if count == 0 {
		count = 5
	}
	songs := make([]TopSong, 0, count)
	for i := range count {
		songs = append(songs, TopSong{
			Name: input.Name + " Song " + string(rune('1'+i)),
		})
	}
	output := TopSongsOutput{Songs: songs}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport nd_get_album_info
func ndGetAlbumInfo() int32 {
	if hasErr, code := checkConfigError(); hasErr {
		return code
	}
	var input AlbumInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}
	output := AlbumInfoOutput{
		Name:        input.Name,
		MBID:        "test-album-mbid-" + input.Name,
		Description: "Description for " + input.Name + " by " + input.Artist,
		URL:         "https://test.example.com/album/" + input.Name,
	}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport nd_get_album_images
func ndGetAlbumImages() int32 {
	if hasErr, code := checkConfigError(); hasErr {
		return code
	}
	var input AlbumInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}
	output := AlbumImagesOutput{
		Images: []ArtistImage{
			{URL: "https://test.example.com/albums/" + input.Name + "/cover.jpg", Size: 500},
		},
	}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

func main() {}
