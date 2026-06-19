package model

import (
	"encoding/xml"
	"fmt"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/log"
)

// lyricParser parses a single lyrics format. It returns an empty list (not an
// error) when the input is well-formed but not its format, so candidates can be
// tried in order. The lang argument is the default language for formats that do
// not carry their own.
type lyricParser func(lang string, contents []byte) (LyricList, error)

func parseLyricsfileBytes(_ string, contents []byte) (LyricList, error) {
	return parseLyricsfile(string(contents))
}

// parseTTMLIfDocument parses TTML only when the content actually looks like a
// TTML document, so content-sniffing plain text or LRC does not run the XML
// decoder (and never logs a spurious parse warning for non-TTML input).
func parseTTMLIfDocument(lang string, contents []byte) (LyricList, error) {
	if !isTTMLDocument(string(contents)) {
		return nil, nil
	}
	return parseTTMLWithDefaultLang(lang, contents)
}

// registry is the single source of truth for the structured formats. Slice
// order is the content-sniff probe order (TTML → SRT → YAML), and each row's
// suffixes drive sidecar dispatch. byContent is used when sniffing — it gates a
// format so, e.g., TTML only runs the XML decoder on something that looks like a
// document; bySuffix parses unconditionally because the file extension already
// declares the format. LRC/plain is not listed: it is the fallback floor for
// both paths.
var registry = []struct {
	suffixes  []string
	bySuffix  lyricParser
	byContent lyricParser
}{
	{[]string{".ttml"}, parseTTMLWithDefaultLang, parseTTMLIfDocument},
	{[]string{".srt"}, parseSRTWithLanguage, parseSRTWithLanguage},
	{[]string{".yaml", ".yml"}, parseLyricsfileBytes, parseLyricsfileBytes},
}

// ParseLyrics is the single entry point for parsing lyrics. A known suffix
// (.ttml/.srt/.yaml/.yml/.lrc) routes to that format's parser; an empty or
// "auto" suffix content-sniffs. In both modes a structured parser that does not
// match falls back to the LRC/plain-text floor — never to another structured
// format.
func ParseLyrics(suffix, lang string, contents []byte) (LyricList, error) {
	if suffix == "" || strings.EqualFold(suffix, "auto") {
		candidates := make([]lyricParser, len(registry))
		for i, r := range registry {
			candidates[i] = r.byContent
		}
		return parseFirstMatch(lang, stripBOM(contents), candidates...)
	}
	for _, r := range registry {
		if slices.ContainsFunc(r.suffixes, func(s string) bool { return strings.EqualFold(s, suffix) }) {
			return parseFirstMatch(lang, contents, r.bySuffix)
		}
	}
	return plainLRC(lang, contents) // .lrc and any unknown suffix
}

// parseFirstMatch tries each candidate in order, returning the first non-empty
// result. When every candidate misses, it falls to the LRC/plain-text floor.
func parseFirstMatch(lang string, contents []byte, candidates ...lyricParser) (LyricList, error) {
	for _, parse := range candidates {
		list, err := parse(lang, contents)
		if err == nil && len(list) > 0 {
			return list, nil
		}
		if err != nil {
			log.Warn("Error parsing lyrics, falling back to plain text", "error", err)
		}
	}
	return plainLRC(lang, contents)
}

func plainLRC(lang string, contents []byte) (LyricList, error) {
	lyric, err := parseLRC(lang, string(contents))
	if err != nil {
		return nil, fmt.Errorf("parsing lyrics: %w", err)
	}
	if lyric == nil || lyric.IsEmpty() {
		return nil, nil
	}
	return LyricList{*lyric}, nil
}

func stripBOM(contents []byte) []byte {
	return []byte(strings.TrimPrefix(string(contents), "\ufeff"))
}

func isTTMLDocument(text string) bool {
	decoder := xml.NewDecoder(strings.NewReader(strings.TrimSpace(text)))
	for {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		if start, ok := token.(xml.StartElement); ok {
			return strings.EqualFold(start.Name.Local, "tt")
		}
	}
}
