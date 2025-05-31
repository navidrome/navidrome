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

type Line struct {
	Start *int64 `structs:"start,omitempty" json:"start,omitempty"`
	Value string `structs:"value"           json:"value"`
}

type Lyrics struct {
	DisplayArtist string `structs:"displayArtist,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string `structs:"displayTitle,omitempty"  json:"displayTitle,omitempty"`
	Lang          string `structs:"lang"                    json:"lang"`
	Line          []Line `structs:"line"                    json:"line"`
	Offset        *int64 `structs:"offset,omitempty"        json:"offset,omitempty"`
	Synced        bool   `structs:"synced"                  json:"synced"`
}

// support the standard [mm:ss.mm], as well as [hh:*] and [*.mmm]
const timeRegexString = `\[([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})(.[0-9]{1,3})?\]`

var (
	// Should either be at the beginning of file, or beginning of line
	syncRegex  = regexp.MustCompile(`(^|\n)\s*` + timeRegexString)
	timeRegex  = regexp.MustCompile(timeRegexString)
	lrcIdRegex = regexp.MustCompile(`\[(ar|ti|offset|lang):([^]]+)]`)
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
					structuredLines = append(structuredLines, Line{
						Start: &timestamps[idx],
						Value: strings.TrimSpace(priorLine),
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
			structuredLines = append(structuredLines, Line{
				Start: &timestamps[idx],
				Value: strings.TrimSpace(priorLine),
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
		Line:          structuredLines,
		Offset:        offset,
		Synced:        synced,
	}
	return &lyrics, nil
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
