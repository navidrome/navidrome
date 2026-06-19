package model

import (
	"encoding/xml"
	"strings"

	"github.com/navidrome/navidrome/log"
)

// ParseEmbedded parses lyrics read from media-file metadata tags. It detects rich
// payloads before falling back to the generic LRC/plain-text parser, because
// text sanitization would otherwise strip TTML XML markup.
func ParseEmbedded(language, text string) (LyricList, error) {
	text = strings.TrimPrefix(text, "\ufeff")

	if isTTMLDocument(text) {
		list, err := parseTTMLWithDefaultLang([]byte(text), language)
		if err == nil && len(list) > 0 {
			return list, nil
		}
		if err != nil {
			log.Warn("Error parsing embedded TTML lyrics, falling back to plain lyrics", "error", err)
		}
	}

	list, err := parseSRTWithLanguage([]byte(text), language)
	if err == nil && len(list) > 0 {
		return list, nil
	}
	if err != nil && strings.Contains(text, "-->") {
		log.Warn("Error parsing embedded SRT lyrics, falling back to plain lyrics", "error", err)
	}

	lyric, err := ToLyrics(language, text)
	if err != nil {
		return nil, err
	}
	if lyric == nil || lyric.IsEmpty() {
		return nil, nil
	}
	return LyricList{*lyric}, nil
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
