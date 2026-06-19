package model

import (
	"encoding/xml"
	"fmt"
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

// bySuffix maps a sidecar file extension to its parser. LRC and unknown
// extensions are absent: they fall straight to the plain-text floor. The suffix
// already declares the format, so TTML parses unconditionally here (a malformed
// .ttml surfaces its error and then the plain-text fallback).
var bySuffix = map[string]lyricParser{
	".ttml": parseTTMLWithDefaultLang,
	".srt":  parseSRTWithLanguage,
	".yaml": parseLyricsfileBytes,
	".yml":  parseLyricsfileBytes,
}

// sniffOrder is the content-sniff candidate order for tag-embedded lyrics and
// plugin responses (no suffix to dispatch on): TTML → SRT → YAML → LRC/plain.
// TTML is gated on looking like a document so plain/LRC text is not run through
// the XML decoder.
var sniffOrder = []lyricParser{parseTTMLIfDocument, parseSRTWithLanguage, parseLyricsfileBytes}

// ParseLyrics is the single entry point for parsing lyrics. A known suffix
// (.ttml/.srt/.yaml/.yml/.lrc) routes to that format's parser; an empty or
// "auto" suffix content-sniffs. In both modes a structured parser that does not
// match falls back to the LRC/plain-text floor — never to another structured
// format.
func ParseLyrics(suffix, lang string, contents []byte) (LyricList, error) {
	if suffix == "" || strings.EqualFold(suffix, "auto") {
		return parseFirstMatch(lang, stripBOM(contents), sniffOrder...)
	}
	if parse, ok := bySuffix[strings.ToLower(suffix)]; ok {
		return parseFirstMatch(lang, contents, parse)
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
