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

// LyricKind enumerates the v2 OpenSubsonic lyric kinds.
type LyricKind string

const (
	LyricKindMain          LyricKind = "main"
	LyricKindTranslation   LyricKind = "translation"
	LyricKindPronunciation LyricKind = "pronunciation"
)

// Cue is a single sub-line timed unit (a word or syllable) inside a CueLine.
// Used for word-level karaoke timing.
type Cue struct {
	Start *int64 `structs:"start,omitempty" json:"start,omitempty"`
	End   *int64 `structs:"end,omitempty"   json:"end,omitempty"`
	Value string `structs:"value"           json:"value"`
}

// CueLine is the canonical v2-shaped representation of a single timed lyric
// line. Index identifies a logical lyrical moment: cuelines that share an
// index are intended to render simultaneously and are disambiguated by AgentID.
// In the absence of overlapping vocals Index equals the cueLine's position in
// the slice.
type CueLine struct {
	Index   int    `structs:"index"                 json:"index"`
	Start   *int64 `structs:"start,omitempty"       json:"start,omitempty"`
	End     *int64 `structs:"end,omitempty"         json:"end,omitempty"`
	Value   string `structs:"value"                 json:"value"`
	AgentID string `structs:"agentId,omitempty"     json:"agentId,omitempty"`
	Cue     []Cue  `structs:"cue,omitempty"         json:"cue,omitempty"`
}

// Agent declares a vocalist referenced by CueLine.AgentID.
type Agent struct {
	ID   string `structs:"id"             json:"id"`
	Name string `structs:"name,omitempty" json:"name,omitempty"`
	Role string `structs:"role,omitempty" json:"role,omitempty"`
}

// Lyrics is the canonical v2-shaped lyrics document. The legacy v1 wire shape
// (Line[]) is derived at response build time by collapsing CueLine[].
type Lyrics struct {
	DisplayArtist string    `structs:"displayArtist,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string    `structs:"displayTitle,omitempty"  json:"displayTitle,omitempty"`
	Lang          string    `structs:"lang"                    json:"lang"`
	Offset        *int64    `structs:"offset,omitempty"        json:"offset,omitempty"`
	Synced        bool      `structs:"synced"                  json:"synced"`
	Kind          LyricKind `structs:"kind,omitempty"          json:"kind,omitempty"`
	Agents        []Agent   `structs:"agents,omitempty"        json:"agents,omitempty"`
	CueLine       []CueLine `structs:"cueLine,omitempty"       json:"cueLine,omitempty"`
}

func (l Lyrics) IsEmpty() bool {
	return len(l.CueLine) == 0
}

// support the standard [mm:ss.mm], as well as [hh:*] and [*.mmm]
const timeRegexString = `\[([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})(.[0-9]{1,3})?\]`

// ELRC inline word-timing markers, e.g. <00:12.45> at the start of a word.
const wordTimeRegexString = `<([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})(.[0-9]{1,3})?>`

var (
	// Should either be at the beginning of file, or beginning of line
	syncRegex     = regexp.MustCompile(`(^|\n)\s*` + timeRegexString)
	timeRegex     = regexp.MustCompile(timeRegexString)
	wordTimeRegex = regexp.MustCompile(wordTimeRegexString)
	lrcIdRegex    = regexp.MustCompile(`\[(ar|ti|offset|lang):([^]]+)]`)
)

// ToLyrics parses a textual lyrics source (plain text, LRC, or ELRC) into the
// canonical Lyrics structure. ELRC inline word-timing markers (`<mm:ss.xx>`)
// are recognized within timestamped lines and produce per-cue word-level data.
// CueLine.End is inferred as the next CueLine's Start; the last CueLine's End
// is left nil.
func ToLyrics(language, text string) (*Lyrics, error) {
	text = str.SanitizeText(text)

	lines := strings.Split(text, "\n")
	cueLines := make([]CueLine, 0, len(lines)*2)

	artist := ""
	title := ""
	var offset *int64 = nil

	synced := syncRegex.MatchString(text)
	priorLine := ""
	priorCues := []Cue(nil)
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
				flushPriorLine(&cueLines, timestamps, priorLine, priorCues)
				timestamps = nil
				priorCues = nil
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
				rest := strings.TrimSpace(line[end:])
				priorLine, priorCues = parseELRCLine(rest)
			}

			validLine = true
		} else {
			cueLines = append(cueLines, CueLine{
				Index: len(cueLines),
				Value: line,
			})
		}
	}

	if validLine {
		flushPriorLine(&cueLines, timestamps, priorLine, priorCues)
	}

	// If there are repeated values, there is no guarantee that they are in order
	// In this case, sort the lyrics by start time and reassign indices.
	if repeated {
		slices.SortFunc(cueLines, func(a, b CueLine) int {
			return cmp.Compare(*a.Start, *b.Start)
		})
		for i := range cueLines {
			cueLines[i].Index = i
		}
	}

	// Infer end-of-line from the next line's start (line-only data).
	for i := 0; i < len(cueLines)-1; i++ {
		if cueLines[i].End == nil && cueLines[i+1].Start != nil {
			cueLines[i].End = cueLines[i+1].Start
		}
	}

	lyrics := Lyrics{
		DisplayArtist: artist,
		DisplayTitle:  title,
		Lang:          language,
		CueLine:       cueLines,
		Offset:        offset,
		Synced:        synced,
	}
	return &lyrics, nil
}

// flushPriorLine emits one CueLine per accumulated timestamp, copying the
// shared text/cue data into each. The last "Repeated" use-case is preserved
// (a single text repeats at multiple timestamps).
func flushPriorLine(cueLines *[]CueLine, timestamps []int64, text string, cues []Cue) {
	trimmed := strings.TrimSpace(text)
	for idx := range timestamps {
		startCopy := timestamps[idx]
		cl := CueLine{
			Index: len(*cueLines),
			Start: &startCopy,
			Value: trimmed,
		}
		// Cues are only meaningful for the first occurrence in a repeat; copy
		// them onto each cueLine so each independent index has its own cue list.
		if len(cues) > 0 {
			cl.Cue = make([]Cue, len(cues))
			copy(cl.Cue, cues)
		}
		*cueLines = append(*cueLines, cl)
	}
}

// parseELRCLine takes the text following a line-level [mm:ss.xx] marker and
// extracts inline ELRC word timestamps `<mm:ss.xx>word `. It returns the
// concatenated text (which equals cueLine.value when cues are present) and
// the list of cues. When no inline markers are present the cues slice is nil
// and the returned text is the input unchanged.
func parseELRCLine(text string) (string, []Cue) {
	matches := wordTimeRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil
	}

	// Anything before the first marker is text without a cue timestamp; we
	// fold it onto the start of the first cue (most common case is no
	// preamble at all).
	var preamble string
	first := matches[0]
	if first[0] > 0 {
		preamble = text[:first[0]]
	}

	cues := make([]Cue, 0, len(matches))
	for i, m := range matches {
		startMs, err := parseTime(text, m)
		if err != nil {
			// Malformed timestamp; treat the rest as plain text.
			return text, nil
		}
		ts := startMs
		var cueValue string
		if i+1 < len(matches) {
			cueValue = text[m[1]:matches[i+1][0]]
		} else {
			cueValue = text[m[1]:]
		}
		if i == 0 && preamble != "" {
			cueValue = preamble + cueValue
		}
		cues = append(cues, Cue{Start: &ts, Value: cueValue})
	}

	// Set each cue's End = next cue's Start. Last cue's End remains nil.
	for i := 0; i < len(cues)-1; i++ {
		cues[i].End = cues[i+1].Start
	}

	// CueLine.Value is the concatenation of cue values per the v2 spec.
	var sb strings.Builder
	for _, c := range cues {
		sb.WriteString(c.Value)
	}
	return sb.String(), cues
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
