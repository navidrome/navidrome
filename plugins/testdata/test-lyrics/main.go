// Test lyrics plugin for Navidrome plugin system integration tests.
package main

import (
	"fmt"

	"github.com/navidrome/navidrome/plugins/pdk/go/lyrics"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

func init() {
	lyrics.Register(&testLyrics{})
}

type testLyrics struct{}

func (t *testLyrics) GetLyrics(input lyrics.GetLyricsRequest) (lyrics.GetLyricsResponse, error) {
	// Check for configured error
	errMsg, hasErr := pdk.GetConfig("error")
	if hasErr && errMsg != "" {
		return lyrics.GetLyricsResponse{}, fmt.Errorf("%s", errMsg)
	}

	// Check if we should omit language (to test default language handling)
	noLang, hasNoLang := pdk.GetConfig("no_lang")
	lang := "eng"
	if hasNoLang && noLang == "true" {
		lang = ""
	}

	// Return test lyrics based on track info
	return lyrics.GetLyricsResponse{
		Lyrics: []lyrics.LyricsText{
			{
				Lang: lang,
				Text: "Test lyrics for " + input.Track.Title + "\nBy " + input.Track.Artist,
			},
		},
	}, nil
}

func main() {}
