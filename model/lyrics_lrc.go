package model

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/str"
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

func parseLRC(language, text string) (*Lyrics, error) {
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
			if len(times) == 0 || times[0][0] != 0 {
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
		Line:          normalizeCueLines(structuredLines),
		Offset:        offset,
		Synced:        synced,
	}
	return &lyrics, nil
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
	var trailingEnd *int64
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
			if i == len(matches)-1 {
				trailingEnd = &timeMs
			}
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

	if trailingEnd != nil && len(cues) > 0 {
		cues[len(cues)-1].End = trailingEnd
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
