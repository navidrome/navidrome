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

	// Return a minimal TTML document to exercise content-sniffing for rich formats.
	format, hasFormat := pdk.GetConfig("format")
	if hasFormat && format == "ttml" {
		const ttml = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng">
    <div>
      <p begin="00:00.000" end="00:01.000">plugin ttml line</p>
    </div>
  </body>
</tt>`
		return lyrics.GetLyricsResponse{
			Lyrics: []lyrics.LyricsText{
				{Lang: "eng", Text: ttml},
			},
		}, nil
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
