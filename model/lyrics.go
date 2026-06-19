package model

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/str"
)

type Cue struct {
	Start     *int64 `structs:"start,omitempty"   json:"start,omitempty"`
	End       *int64 `structs:"end,omitempty"     json:"end,omitempty"`
	Value     string `structs:"value"             json:"value"`
	ByteStart int    `structs:"byteStart"         json:"byteStart"`
	ByteEnd   int    `structs:"byteEnd"           json:"byteEnd"`
	AgentID   string `structs:"agentId,omitempty" json:"agentId,omitempty"`
}

type Agent struct {
	ID   string `structs:"id"             json:"id"`
	Role string `structs:"role"           json:"role"`
	Name string `structs:"name,omitempty" json:"name,omitempty"`
}

type Line struct {
	Start *int64 `structs:"start,omitempty" json:"start,omitempty"`
	End   *int64 `structs:"end,omitempty"   json:"end,omitempty"`
	Value string `structs:"value"           json:"value"`
	Cue   []Cue  `structs:"cue,omitempty"   json:"cue,omitempty"`
}

type Lyrics struct {
	DisplayArtist string  `structs:"displayArtist,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string  `structs:"displayTitle,omitempty"  json:"displayTitle,omitempty"`
	Kind          string  `structs:"kind,omitempty"          json:"kind,omitempty"`
	Lang          string  `structs:"lang"                    json:"lang"`
	Agents        []Agent `structs:"agents,omitempty"       json:"agents,omitempty"`
	Line          []Line  `structs:"line"                    json:"line"`
	Offset        *int64  `structs:"offset,omitempty"        json:"offset,omitempty"`
	Synced        bool    `structs:"synced"                  json:"synced"`
}

// Lyric kinds, as defined by the OpenSubsonic songLyrics v2 contract. These are
// the canonical wire values; keep them in sync with the spec.
const (
	LyricKindMain          = "main"
	LyricKindTranslation   = "translation"
	LyricKindPronunciation = "pronunciation"
)

// support the standard [mm:ss.mm], as well as [hh:*] and [*.mmm]
const timeRegexString = `\[([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})(\.[0-9]{1,3})?\]`

var (
	// Should either be at the beginning of file, or beginning of line
	syncRegex  = regexp.MustCompile(`(^|\n)\s*` + timeRegexString)
	timeRegex  = regexp.MustCompile(timeRegexString)
	lrcIdRegex = regexp.MustCompile(`\[(ar|ti|offset|lang):([^]]+)]`)

	// Enhanced LRC: inline word-level timing markers like <00:12.34>
	enhancedLRCTimeString = `<([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})(\.[0-9]{1,3})?>`
	enhancedLRCRegex      = regexp.MustCompile(enhancedLRCTimeString)
)

func (l Lyrics) IsEmpty() bool {
	return len(l.Line) == 0
}

// IsMainKind reports whether the lyric is the main track. A blank kind is an
// untyped (single-track) lyric, which the contract treats as main.
func (l Lyrics) IsMainKind() bool {
	return l.EffectiveKind() == LyricKindMain
}

// EffectiveKind returns the lyric kind, defaulting to LyricKindMain when blank.
// A blank kind means an untyped (single-track) lyric, which the contract treats
// as main.
func (l Lyrics) EffectiveKind() string {
	if strings.TrimSpace(l.Kind) == "" {
		return LyricKindMain
	}
	return l.Kind
}

func ToLyrics(language, text string) (*Lyrics, error) {
	text = str.SanitizeText(text)

	lines := strings.Split(text, "\n")
	structuredLines := make([]Line, 0, len(lines)*2)

	artist := ""
	title := ""
	var offset *int64 = nil

	synced := syncRegex.MatchString(text)
	priorLine := ""
	validLine := false
	repeated := false
	var timestamps []int64

	for _, line := range lines {
		line := strings.TrimSpace(line)
		if line == "" {
			if validLine {
				priorLine += "\n"
			}
			continue
		}
		var text string
		var time *int64 = nil

		if synced {
			idTag := lrcIdRegex.FindStringSubmatch(line)
			if idTag != nil {
				switch idTag[1] {
				case "ar":
					artist = str.SanitizeText(strings.TrimSpace(idTag[2]))
				case "lang":
					language = str.SanitizeText(strings.TrimSpace(idTag[2]))
				case "offset":
					{
						off, err := strconv.ParseInt(strings.TrimSpace(idTag[2]), 10, 64)
						if err != nil {
							log.Warn("Error parsing offset", "offset", idTag[2], "error", err)
						} else {
							offset = &off
						}
					}
				case "ti":
					title = str.SanitizeText(strings.TrimSpace(idTag[2]))
				}

				continue
			}

			times := timeRegex.FindAllStringSubmatchIndex(line, -1)
			if len(times) > 1 {
				repeated = true
			}

			// The second condition is for when there is a timestamp in the middle of
			// a line (after any text)
			if times == nil || times[0][0] != 0 {
				if validLine {
					priorLine += "\n" + line
				}
				continue
			}

			if validLine {
				value, baseCues := parseEnhancedLine(priorLine)
				for idx := range timestamps {
					startCopy := timestamps[idx]
					structuredLines = append(structuredLines, Line{
						Start: &startCopy,
						Value: value,
						Cue:   shiftELRCCues(baseCues, timestamps[idx]-timestamps[0]),
					})
				}
				timestamps = nil
			}

			end := 0

			// [fullStart, fullEnd, hourStart, hourEnd, minStart, minEnd, secStart, secEnd, msStart, msEnd]
			for _, match := range times {
				// for multiple matches, we need to check that later matches are not
				// in the middle of the string
				if end != 0 {
					middle := strings.TrimSpace(line[end:match[0]])
					if middle != "" {
						break
					}
				}

				end = match[1]
				timeInMillis, err := parseTime(line, match)
				if err != nil {
					return nil, err
				}

				timestamps = append(timestamps, timeInMillis)
			}

			if end >= len(line) {
				priorLine = ""
			} else {
				priorLine = strings.TrimSpace(line[end:])
			}

			validLine = true
		} else {
			text = line
			structuredLines = append(structuredLines, Line{
				Start: time,
				Value: text,
			})
		}
	}

	if validLine {
		value, baseCues := parseEnhancedLine(priorLine)
		for idx := range timestamps {
			startCopy := timestamps[idx]
			structuredLines = append(structuredLines, Line{
				Start: &startCopy,
				Value: value,
				Cue:   shiftELRCCues(baseCues, timestamps[idx]-timestamps[0]),
			})
		}
	}

	// If there are repeated values, there is no guarantee that they are in order
	// In this, case, sort the lyrics by start time
	if repeated {
		slices.SortFunc(structuredLines, func(a, b Line) int {
			return cmp.Compare(*a.Start, *b.Start)
		})
	}

	lyrics := Lyrics{
		DisplayArtist: artist,
		DisplayTitle:  title,
		Lang:          language,
		Line:          NormalizeCueLines(structuredLines),
		Offset:        offset,
		Synced:        synced,
	}
	return &lyrics, nil
}

// ParseLyricsFile parses a sidecar lyrics file, dispatching on its extension to
// the matching format parser. Unknown extensions fall back to the generic
// LRC/plain-text parser. It is the single owner of the suffix→parser mapping,
// mirroring [ParseEmbedded] for tag-embedded lyrics.
func ParseLyricsFile(suffix string, contents []byte) (LyricList, error) {
	var list LyricList
	var err error
	switch {
	case strings.EqualFold(suffix, ".ttml"):
		list, err = ParseTTML(contents)
	case strings.EqualFold(suffix, ".srt"):
		list, err = ParseSRT(contents)
	case strings.EqualFold(suffix, ".yaml"), strings.EqualFold(suffix, ".yml"):
		list, err = ParseLyricsfile(string(contents))
	default:
		var lyric *Lyrics
		lyric, err = ToLyrics("xxx", string(contents))
		if lyric != nil {
			list = LyricList{*lyric}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("parsing %s lyrics: %w", strings.TrimPrefix(suffix, "."), err)
	}
	return list, nil
}

// parseEnhancedLine extracts word-level timing cues from Enhanced LRC inline markers
// and computes UTF-8 byte offsets against the final stripped line value.
func parseEnhancedLine(text string) (string, []Cue) {
	matches := enhancedLRCRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(text), nil
	}

	type segment struct {
		start    int64
		rawStart int
		rawEnd   int
	}

	segments := make([]segment, 0, len(matches))
	var rawValue strings.Builder
	for i, match := range matches {
		timeMs, err := parseTime(
			// Rewrite <...> as [...] so parseTime can handle it with the same logic
			"["+text[match[0]+1:match[1]-1]+"]",
			// Adjust match indices to point into our rewritten string (need start/end pairs for each group)
			[]int{
				0, match[1] - match[0],
				adjustGroup(match, 2), adjustGroup(match, 3),
				adjustGroup(match, 4), adjustGroup(match, 5),
				adjustGroup(match, 6), adjustGroup(match, 7),
				adjustGroup(match, 8), adjustGroup(match, 9),
			},
		)
		if err != nil {
			continue
		}

		// Text runs from after this marker to the start of the next marker (or end of string)
		textStart := match[1]
		var textEnd int
		if i+1 < len(matches) {
			textEnd = matches[i+1][0]
		} else {
			textEnd = len(text)
		}

		word := text[textStart:textEnd]
		if word == "" {
			continue
		}

		rawStart := rawValue.Len()
		rawValue.WriteString(word)
		segments = append(segments, segment{
			start:    timeMs,
			rawStart: rawStart,
			rawEnd:   rawValue.Len(),
		})
	}

	if len(segments) == 0 {
		return strings.TrimSpace(stripEnhancedMarkers(text)), nil
	}

	finalRaw := rawValue.String()
	leftTrimBytes := len(finalRaw) - len(strings.TrimLeftFunc(finalRaw, unicode.IsSpace))
	rightTrimBytes := len(finalRaw) - len(strings.TrimRightFunc(finalRaw, unicode.IsSpace))
	trimmedEnd := len(finalRaw) - rightTrimBytes
	if trimmedEnd < leftTrimBytes {
		trimmedEnd = leftTrimBytes
	}

	cues := make([]Cue, 0, len(segments))
	for _, seg := range segments {
		start := seg.start
		byteStart := max(seg.rawStart, leftTrimBytes)
		byteEnd := min(seg.rawEnd, trimmedEnd)
		if byteStart >= byteEnd {
			continue
		}

		cues = append(cues, Cue{
			Start:     &start,
			Value:     finalRaw[byteStart:byteEnd],
			ByteStart: byteStart - leftTrimBytes,
			ByteEnd:   byteEnd - leftTrimBytes - 1,
		})
	}

	return strings.TrimSpace(finalRaw), cues
}

// adjustGroup remaps a capture group index from the original match to our rewritten "[...]" string.
// The rewrite shifts by -1 (removed '<', added '[') so positions within the brackets stay the same.
func adjustGroup(match []int, groupIdx int) int {
	orig := match[groupIdx]
	if orig == -1 {
		return -1
	}
	// Offset is: original position minus the position of '<' in the original, plus 1 for '['
	return orig - match[0]
}

// stripEnhancedMarkers removes all <mm:ss.mm> inline markers from text,
// returning the plain lyric text.
func stripEnhancedMarkers(text string) string {
	return enhancedLRCRegex.ReplaceAllString(text, "")
}

// shiftELRCCues returns a deep copy of baseCues with each cue's Start/End
// timestamps shifted by offsetMs. Inline ELRC word markers parse to absolute
// timestamps anchored at the line's first occurrence, so repeated-line LRC
// inputs of the form `[t0][t1]...` must shift the cues by (t1-t0) for the
// second occurrence to point at the correct moment. Returned *int64 pointers
// are freshly allocated so the input slice is never aliased into the result.
func shiftELRCCues(baseCues []Cue, offsetMs int64) []Cue {
	if len(baseCues) == 0 {
		return nil
	}
	out := make([]Cue, len(baseCues))
	for i, c := range baseCues {
		out[i] = c
		if c.Start != nil {
			s := *c.Start + offsetMs
			out[i].Start = &s
		}
		if c.End != nil {
			e := *c.End + offsetMs
			out[i].End = &e
		}
	}
	return out
}

func parseTime(line string, match []int) (int64, error) {
	var hours, millis int64
	var err error

	hourStart := match[2]
	if hourStart != -1 {
		// subtract 1 because group has : at the end
		hourEnd := match[3] - 1
		hours, err = strconv.ParseInt(line[hourStart:hourEnd], 10, 64)
		if err != nil {
			return 0, err
		}
	}

	minutes, err := strconv.ParseInt(line[match[4]:match[5]], 10, 64)
	if err != nil {
		return 0, err
	}

	sec, err := strconv.ParseInt(line[match[6]:match[7]], 10, 64)
	if err != nil {
		return 0, err
	}

	msStart := match[8]
	if msStart != -1 {
		msEnd := match[9]
		// +1 offset since this capture group contains .
		millis, err = strconv.ParseInt(line[msStart+1:msEnd], 10, 64)
		if err != nil {
			return 0, err
		}

		length := msEnd - msStart

		if length == 3 {
			millis *= 10
		} else if length == 2 {
			millis *= 100
		}
	}

	timeInMillis := (((((hours * 60) + minutes) * 60) + sec) * 1000) + millis
	return timeInMillis, nil
}

type LyricList []Lyrics

// Main returns the main-kind lyric, falling back to the first entry so untyped
// lyrics still resolve. The bool is false only when the list is empty. It is
// used to surface a single lyric through the plain-text legacy getLyrics
// endpoint, which has no notion of translation/pronunciation tracks.
func (ll LyricList) Main() (Lyrics, bool) {
	if len(ll) == 0 {
		return Lyrics{}, false
	}
	for _, l := range ll {
		if l.IsMainKind() {
			return l, true
		}
	}
	return ll[0], true
}

func NormalizeLyrics(lyrics Lyrics) Lyrics {
	lyrics.Line = NormalizeCueLines(lyrics.Line)
	if len(lyrics.Agents) == 0 {
		lyrics.Agents = nil
	}
	return lyrics
}

func NormalizeCueLines(lines []Line) []Line {
	if len(lines) == 0 {
		return lines
	}

	normalized := make([]Line, len(lines))
	copy(normalized, lines)

	for i := range normalized {
		if len(normalized[i].Cue) > 0 {
			normalized[i].Cue = slices.Clone(normalized[i].Cue)
		}

		var fallbackEnd *int64
		if normalized[i].End != nil {
			v := *normalized[i].End
			fallbackEnd = &v
		} else if i+1 < len(normalized) && normalized[i+1].Start != nil {
			v := *normalized[i+1].Start
			fallbackEnd = &v
		}

		normalized[i] = normalizeCueLine(normalized[i], fallbackEnd)
	}

	return normalized
}

func NormalizeLineTiming(line Line) Line {
	if len(line.Cue) == 0 {
		return line
	}

	var earliestStart *int64
	var latestEnd *int64
	for i := range line.Cue {
		token := line.Cue[i]
		if token.Start != nil {
			if earliestStart == nil || *token.Start < *earliestStart {
				v := *token.Start
				earliestStart = &v
			}
		}

		candidateEnd := token.End
		if candidateEnd == nil {
			candidateEnd = token.Start
		}
		if candidateEnd != nil {
			if latestEnd == nil || *candidateEnd > *latestEnd {
				v := *candidateEnd
				latestEnd = &v
			}
		}
	}

	if line.Start == nil && earliestStart != nil {
		v := *earliestStart
		line.Start = &v
	}
	if line.End == nil && latestEnd != nil {
		v := *latestEnd
		line.End = &v
	}
	return line
}

func normalizeCueLine(line Line, fallbackEnd *int64) Line {
	if len(line.Cue) == 0 {
		return line
	}
	line.Cue = NormalizeCueEnds(line.Cue, fallbackEnd)
	return NormalizeLineTiming(line)
}

// NormalizeCueEnds resolves missing cue end times within a single ordered cue
// group: each end is filled from the next cue's start, then from fallbackEnd,
// and is clamped so it never precedes the cue's own start nor overruns the next
// cue. End times are all-or-none — if any cue still lacks an end afterwards, all
// ends in the group are cleared. The input slice is never mutated.
func NormalizeCueEnds(cues []Cue, fallbackEnd *int64) []Cue {
	if len(cues) == 0 {
		return cues
	}

	out := slices.Clone(cues)
	for i := range out {
		end := out[i].End
		if end == nil {
			if i+1 < len(out) && out[i+1].Start != nil {
				end = out[i+1].Start
			} else {
				end = fallbackEnd
			}
		}
		if end != nil && i+1 < len(out) && out[i+1].Start != nil && *end > *out[i+1].Start {
			end = out[i+1].Start
		}
		if end != nil && out[i].Start != nil && *end < *out[i].Start {
			end = out[i].Start
		}
		out[i].End = gg.Clone(end)
	}

	for i := range out {
		if out[i].End == nil {
			for j := range out {
				out[j].End = nil
			}
			break
		}
	}
	return out
}
