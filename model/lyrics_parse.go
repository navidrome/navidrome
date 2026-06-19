package model

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/log"
)

// ParseLyrics is the single entry point for parsing lyrics. When suffix names a
// known format (.ttml/.srt/.yaml/.yml/.lrc) it routes to that parser, falling
// back to plain text on failure. When suffix is empty or "auto" it content-sniffs
// (TTML → SRT → YAML/Lyricsfile → LRC/plain) — used for tag-embedded lyrics and
// plugin responses.
func ParseLyrics(suffix, lang string, contents []byte) (LyricList, error) {
	switch {
	case suffix == "" || strings.EqualFold(suffix, "auto"):
		return sniffLyrics(lang, contents)
	case strings.EqualFold(suffix, ".ttml"):
		return parseWithPlainFallback(lang, contents, parseTTMLKnown(lang))
	case strings.EqualFold(suffix, ".srt"):
		return parseWithPlainFallback(lang, contents, parseSRTKnown(lang))
	case strings.EqualFold(suffix, ".yaml"), strings.EqualFold(suffix, ".yml"):
		return parseWithPlainFallback(lang, contents, func(c []byte) (LyricList, error) {
			return parseLyricsfile(string(c))
		})
	default: // .lrc and any unknown suffix: LRC/plain is the floor
		return plainLRC(lang, contents)
	}
}

// parseWithPlainFallback runs a known-format parser; on error or empty result it
// falls back to the LRC/plain floor (never another structured format).
func parseWithPlainFallback(lang string, contents []byte, parse func([]byte) (LyricList, error)) (LyricList, error) {
	list, err := parse(contents)
	if err == nil && len(list) > 0 {
		return list, nil
	}
	if err != nil {
		log.Warn("Error parsing lyrics by suffix, falling back to plain text", "error", err)
	}
	return plainLRC(lang, contents)
}

func parseTTMLKnown(lang string) func([]byte) (LyricList, error) {
	return func(c []byte) (LyricList, error) { return parseTTMLWithDefaultLang(c, lang) }
}

func parseSRTKnown(lang string) func([]byte) (LyricList, error) {
	return func(c []byte) (LyricList, error) { return parseSRTWithLanguage(c, lang) }
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

// sniffLyrics detects the format from content. Order: TTML → SRT → YAML → LRC/plain.
func sniffLyrics(lang string, contents []byte) (LyricList, error) {
	text := strings.TrimPrefix(string(contents), "\ufeff")

	if isTTMLDocument(text) {
		list, err := parseTTMLWithDefaultLang([]byte(text), lang)
		if err == nil && len(list) > 0 {
			return list, nil
		}
		if err != nil {
			log.Warn("Error parsing embedded TTML lyrics, falling back to plain lyrics", "error", err)
		}
	}

	list, err := parseSRTWithLanguage([]byte(text), lang)
	if err == nil && len(list) > 0 {
		return list, nil
	}
	if err != nil && strings.Contains(text, "-->") {
		log.Warn("Error parsing embedded SRT lyrics, falling back to plain lyrics", "error", err)
	}

	if yamlList, yErr := parseLyricsfile(text); yErr == nil && len(yamlList) > 0 {
		return yamlList, nil
	}

	return plainLRC(lang, []byte(text))
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
