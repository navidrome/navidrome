// Test lyrics plugin for Navidrome plugin system integration tests.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/navidrome/navidrome/plugins/pdk/go/lyrics"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

func init() {
	lyrics.Register(&testLyrics{})
}

type testLyrics struct{}

func (t *testLyrics) GetLyrics(input lyrics.GetLyricsRequest) (lyrics.GetLyricsResponse, error) {
	errMsg, hasErr := pdk.GetConfig("error")
	if hasErr && errMsg != "" {
		return lyrics.GetLyricsResponse{}, fmt.Errorf("%s", errMsg)
	}

	var source *lyrics.LyricsSource
	provider, hasProvider := pdk.GetConfig("source_provider")
	sourceFormat, hasSourceFormat := pdk.GetConfig("source_format")
	if hasProvider || hasSourceFormat {
		source = &lyrics.LyricsSource{Provider: provider, Format: sourceFormat}
	}

	// Config-provided variants let tests exercise multi-source responses: each
	// entry carries its own source when it names a provider or format, and the
	// response-level source stays the default for the rest.
	if raw, hasVariants := pdk.GetConfig("variants"); hasVariants && raw != "" {
		var variants []struct {
			Lang     string `json:"lang"`
			Text     string `json:"text"`
			Provider string `json:"provider"`
			Format   string `json:"format"`
		}
		if err := json.Unmarshal([]byte(raw), &variants); err != nil {
			return lyrics.GetLyricsResponse{}, fmt.Errorf("decode variants config: %w", err)
		}
		if len(variants) > 0 {
			texts := make([]lyrics.LyricsText, len(variants))
			for i, variant := range variants {
				texts[i] = lyrics.LyricsText{Lang: variant.Lang, Text: variant.Text}
				if variant.Provider != "" || variant.Format != "" {
					texts[i].Source = &lyrics.LyricsSource{Provider: variant.Provider, Format: variant.Format}
				}
			}
			return lyrics.GetLyricsResponse{Lyrics: texts, Source: source}, nil
		}
	}

	// Config-selected format lets tests exercise the adapter's content-sniffing per format.
	format, hasFormat := pdk.GetConfig("format")
	if hasFormat {
		var text string
		var lang string
		switch format {
		case "ttml":
			lang = "eng"
			text = `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng">
    <div>
      <p begin="00:00.000" end="00:01.000">plugin ttml line</p>
    </div>
  </body>
</tt>`
		case "srt":
			lang = "eng"
			text = "1\n00:00:01,000 --> 00:00:02,000\nplugin srt line\n"
		case "yaml":
			lang = "eng"
			text = "version: \"1.0\"\nmetadata:\n  language: eng\nlines:\n  - text: \"plugin yaml line\"\n    start_ms: 1000\n"
		case "lrc":
			lang = "eng"
			text = "[00:01.00]plugin lrc line"
		case "plain":
			lang = "eng"
			text = "plugin plain line"
		}
		if text != "" {
			return lyrics.GetLyricsResponse{
				Lyrics: []lyrics.LyricsText{{Lang: lang, Text: text}},
				Source: source,
			}, nil
		}
	}

	// Check if we should omit language (to test default language handling)
	noLang, hasNoLang := pdk.GetConfig("no_lang")
	lang := "eng"
	if hasNoLang && noLang == "true" {
		lang = ""
	}

	// Return test lyrics based on track info
	return lyrics.GetLyricsResponse{
		Source: source,
		Lyrics: []lyrics.LyricsText{
			{
				Lang: lang,
				Text: "Test lyrics for " + input.Track.Title + "\nBy " + input.Track.Artist,
			},
		},
	}, nil
}

func main() {}
