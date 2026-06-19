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

// lyricFormats lists the structured formats. Each parser is self-skipping: it
// returns an empty list (not an error) when the content is not its format, so it
// is safe to try in order. Slice order is the content-sniff probe order
// (TTML → SRT → YAML); each row's suffixes drive sidecar dispatch. LRC/plain is
// not listed — it is the fallback floor for both paths.
var lyricFormats = []struct {
	suffixes []string
	parse    lyricParser
}{
	{[]string{".ttml"}, parseTTMLWithDefaultLang},
	{[]string{".srt"}, parseSRTWithLanguage},
	{[]string{".yaml", ".yml"}, parseLyricsfileBytes},
}

// ParseLyrics is the single entry point for parsing lyrics. A known suffix
// (.ttml/.srt/.yaml/.yml/.lrc) routes to that format's parser; an empty or
// "auto" suffix content-sniffs by trying each format in order. In both modes a
// structured parser that does not match falls back to the LRC/plain-text floor —
// never to another structured format.
func ParseLyrics(suffix, lang string, contents []byte) (LyricList, error) {
	// Strip a leading BOM once here so every parser sees clean bytes, regardless
	// of which caller (file read, embedded tag, plugin, DB string) supplied them.
	contents = stripBOM(contents)

	if suffix == "" || strings.EqualFold(suffix, "auto") {
		candidates := make([]lyricParser, len(lyricFormats))
		for i, f := range lyricFormats {
			candidates[i] = f.parse
		}
		return parseFirstMatch(lang, contents, candidates...)
	}
	for _, f := range lyricFormats {
		if slices.ContainsFunc(f.suffixes, func(s string) bool { return strings.EqualFold(s, suffix) }) {
			return parseFirstMatch(lang, contents, f.parse)
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
