package model

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/log"
)

// lyricParser returns an empty list (not an error) when the input is not its
// format, so parsers can be tried in order. lang is the default for formats that
// do not carry their own.
type lyricParser func(lang string, contents []byte) (LyricList, error)

// lyricFormats is the structured formats in content-sniff probe order; each
// row's suffixes drive sidecar dispatch. LRC/plain is the unlisted fallback floor.
var lyricFormats = []struct {
	suffixes []string
	parse    lyricParser
}{
	{[]string{".ttml"}, parseTTML},
	{[]string{".srt"}, parseSRT},
	{[]string{".yaml", ".yml"}, parseLyricsfile},
}

// ParseLyrics is the single entry point for parsing lyrics. A known suffix routes
// to that format's parser; an empty or "auto" suffix content-sniffs. Either way,
// a structured parser that does not match falls back to the LRC/plain-text floor.
//
// Parse failures are logged through ctx; callers that know the source should
// attach it for attribution, e.g. log.NewContext(ctx, "file", path).
func ParseLyrics(ctx context.Context, suffix, lang string, contents []byte) (LyricList, error) {
	contents = stripBOM(contents)
	suffix = strings.ToLower(suffix)
	sniff := suffix == "" || suffix == "auto"

	// Sniffing tries every format in order; a known suffix selects just its own.
	// Unmatched suffixes leave no candidates, so parseFirstMatch falls to plain.
	candidates := make([]lyricParser, 0, len(lyricFormats))
	for _, f := range lyricFormats {
		if sniff || slices.Contains(f.suffixes, suffix) {
			candidates = append(candidates, f.parse)
		}
	}
	return parseFirstMatch(ctx, sniff, lang, contents, candidates...)
}

func parseFirstMatch(ctx context.Context, sniff bool, lang string, contents []byte, candidates ...lyricParser) (LyricList, error) {
	for _, parse := range candidates {
		list, err := parse(lang, contents)
		if err == nil && len(list) > 0 {
			return list, nil
		}
		if err != nil {
			// While sniffing, a probe rejecting content it does not own is expected
			// control flow, so keep it at trace. A failure under an explicit suffix
			// means the declared format is malformed and deserves a warning.
			if sniff {
				log.Trace(ctx, "Lyrics probe did not match, trying next format", err)
			} else {
				log.Warn(ctx, "Error parsing lyrics, falling back to plain text", err)
			}
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
	return bytes.TrimPrefix(contents, []byte("\ufeff"))
}
