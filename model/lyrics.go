package model

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/str"
)

type Cue struct {
	Start   *int64 `structs:"start,omitempty"   json:"start,omitempty"`
	End     *int64 `structs:"end,omitempty"     json:"end,omitempty"`
	Value   string `structs:"value"             json:"value"`
	AgentID string `structs:"agentId,omitempty" json:"agentId,omitempty"`
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

// support the standard [mm:ss.mm], as well as [hh:*] and [*.mmm]
const timeRegexString = `\[([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})(.[0-9]{1,3})?\]`

var (
	// Should either be at the beginning of file, or beginning of line
	syncRegex  = regexp.MustCompile(`(^|\n)\s*` + timeRegexString)
	timeRegex  = regexp.MustCompile(timeRegexString)
	lrcIdRegex = regexp.MustCompile(`\[(ar|ti|offset|lang):([^]]+)]`)

	// Enhanced LRC: inline word-level timing markers like <00:12.34>
	enhancedLRCTimeString = `<([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})(.[0-9]{1,3})?>`
	enhancedLRCRegex      = regexp.MustCompile(enhancedLRCTimeString)
)

func (l Lyrics) IsEmpty() bool {
	return len(l.Line) == 0
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
				for idx := range timestamps {
					cues := parseEnhancedCues(priorLine)
					value := priorLine
					if cues != nil {
						value = stripEnhancedMarkers(value)
					}
					structuredLines = append(structuredLines, Line{
						Start: &timestamps[idx],
						Value: strings.TrimSpace(value),
						Cue:   cues,
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
		for idx := range timestamps {
			cues := parseEnhancedCues(priorLine)
			value := priorLine
			if cues != nil {
				value = stripEnhancedMarkers(value)
			}
			structuredLines = append(structuredLines, Line{
				Start: &timestamps[idx],
				Value: strings.TrimSpace(value),
				Cue:   cues,
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

// parseEnhancedCues extracts word-level timing cues from Enhanced LRC inline markers.
// Format: <mm:ss.mm>word <mm:ss.mm>word ...
// Returns nil if no inline markers are found.
func parseEnhancedCues(text string) []Cue {
	matches := enhancedLRCRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	type segment struct {
		start int64
		text  string
	}

	segments := make([]segment, 0, len(matches))
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
		segments = append(segments, segment{start: timeMs, text: word})
	}

	if len(segments) == 0 {
		return nil
	}

	cues := make([]Cue, len(segments))
	for i, seg := range segments {
		start := seg.start
		cues[i] = Cue{
			Start: &start,
			Value: seg.text,
		}
	}
	return cues
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

	for i := range line.Cue {
		if line.Cue[i].End != nil {
			continue
		}

		if i+1 < len(line.Cue) && line.Cue[i+1].Start != nil {
			v := *line.Cue[i+1].Start
			line.Cue[i].End = &v
			continue
		}

		if fallbackEnd != nil {
			v := *fallbackEnd
			line.Cue[i].End = &v
		}
	}

	for i := range line.Cue {
		if line.Cue[i].End == nil {
			line.Cue = clearCueEnds(line.Cue)
			return NormalizeLineTiming(line)
		}
	}

	return NormalizeLineTiming(line)
}

func clearCueEnds(cues []Cue) []Cue {
	normalized := make([]Cue, len(cues))
	copy(normalized, cues)
	for i := range normalized {
		normalized[i].End = nil
	}
	return normalized
}
