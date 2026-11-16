package model

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/log"
)

// lyricParser parses content already claimed by its format. A nil list with no
// error means a recognized, valid document that contains no lyrics.
type lyricParser func(lang string, contents []byte) (LyricList, error)

var errLyricsFormatMismatch = errors.New("lyrics format mismatch")

type lyricFormat struct {
	name     string
	suffixes []string
	claims   func([]byte) bool
	parse    lyricParser
}

// lyricFormats is the structured formats in content-sniff probe order; each
// row's suffixes drive sidecar dispatch. LRC/plain is the unlisted fallback floor.
var lyricFormats = []lyricFormat{
	{name: "TTML", suffixes: []string{".ttml"}, claims: claimsTTML, parse: parseTTML},
	{name: "SRT", suffixes: []string{".srt"}, claims: claimsSRT, parse: parseSRT},
	{name: "Lyricsfile", suffixes: []string{".yaml", ".yml"}, claims: claimsLyricsfile, parse: parseLyricsfile},
}

var (
	ttmlRootPrefixRegex = regexp.MustCompile(`(?is)^\s*(?:<\?xml\b[^>]*\?>\s*)?(?:<!--.*?-->\s*)*<(?:[[:alpha:]_][[:alnum:]_.-]*:)?tt(?:\s|/?>)`)
	srtClaimRegex       = regexp.MustCompile(`(?m)^\s*(?:\d+\s*\n\s*)?\d{1,2}:\d{2}:\d{2}[,.]\d{1,3}\s*-->\s*\d{1,2}:\d{2}:\d{2}[,.]\d{1,3}(?:\s|$)`)
	lyricsfileRegex     = regexp.MustCompile(`(?mi)^\s*["']?version["']?\s*:\s*["']?1\.0["']?\s*(?:#.*)?$`)
)

// ParseLyrics is the single entry point for parsing lyrics. A known suffix routes
// to that format's parser; an empty or "auto" suffix content-sniffs. Explicit
// structured suffixes are strict: malformed or mismatched structured content is
// returned as an error so a source resolver can continue to its next source.
//
// Parse failures are logged through ctx; callers that know the source should
// attach it for attribution, e.g. log.NewContext(ctx, "file", path).
func ParseLyrics(ctx context.Context, suffix, lang string, contents []byte) (LyricList, error) {
	contents = stripBOM(contents)
	suffix = strings.ToLower(suffix)
	sniff := suffix == "" || suffix == "auto"

	// Sniffing tries every structured format in order. A known structured suffix
	// selects one strict parser; LRC/text and unknown textual suffixes retain the
	// longstanding LRC/plain fallback.
	candidates := make([]lyricFormat, 0, len(lyricFormats))
	for _, f := range lyricFormats {
		if sniff || slices.Contains(f.suffixes, suffix) {
			candidates = append(candidates, f)
		}
	}

	if !sniff && len(candidates) > 0 {
		format := candidates[0]
		list, err := parseClaimedLyrics(format, lang, contents)
		if errors.Is(err, errLyricsFormatMismatch) {
			err = fmt.Errorf("declared %s lyrics do not match the format: %w", format.name, err)
		}
		if err != nil {
			log.Warn(ctx, "Error parsing declared lyrics", "format", format.name, err)
		}
		return list, err
	}

	return parseFirstMatch(ctx, lang, contents, candidates...)
}

func parseFirstMatch(ctx context.Context, lang string, contents []byte, candidates ...lyricFormat) (LyricList, error) {
	for _, format := range candidates {
		list, err := parseClaimedLyrics(format, lang, contents)
		if errors.Is(err, errLyricsFormatMismatch) {
			log.Trace(ctx, "Lyrics probe did not match, trying next format", "format", format.name)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("parsing claimed %s lyrics: %w", format.name, err)
		}
		// A claimed, valid-empty document owns the content and deliberately stops
		// sniffing instead of being reinterpreted as another format or plain text.
		return list, nil
	}
	return plainLRC(lang, contents)
}

func parseClaimedLyrics(format lyricFormat, lang string, contents []byte) (LyricList, error) {
	if !format.claims(contents) {
		return nil, errLyricsFormatMismatch
	}
	list, err := format.parse(lang, contents)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func claimsTTML(contents []byte) bool {
	trimmed := bytes.TrimSpace(contents)
	if len(trimmed) == 0 || trimmed[0] != '<' {
		return false
	}
	return isTTMLDocument(contents) || ttmlRootPrefixRegex.Match(contents)
}

func claimsSRT(contents []byte) bool {
	raw := bytes.ReplaceAll(contents, []byte("\r\n"), []byte("\n"))
	raw = bytes.ReplaceAll(raw, []byte("\r"), []byte("\n"))
	return srtClaimRegex.Match(raw)
}

func claimsLyricsfile(contents []byte) bool {
	if lyricsfileRegex.Match(contents) {
		return true
	}
	return hasLyricsfileVersion(contents)
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
