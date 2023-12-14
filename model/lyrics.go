package model

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils"
)

type Line struct {
	Start *int64 `structs:"start,omitempty" xml:"start,attr,omitempty" json:"start,omitempty"`
	Value string `structs:"value"           xml:"value"                json:"value"`
}

type Lyric struct {
	DisplayArtist string `structs:"displayArtist,omitempty" xml:"displayArtist,attr,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string `structs:"displayTitle,omitempty"  xml:"displayTitle,attr,omitempty"  json:"displayTitle,omitempty"`
	Lang          string `structs:"lang"                    xml:"lang,attr"                    json:"lang"`
	Line          []Line `structs:"line"                    xml:"line"                         json:"line"`
	Offset        *int64 `structs:"offset,omitempty"        xml:"offset,attr,omitempty"        json:"offset,omitempty"`
	Synced        bool   `structs:"synced"                  xml:"synced,attr"                  json:"synced"`
}

// support the standard [mm:ss.mm], as well as [hh:*] and [*.mmm]
const timeRegexString = `\[([0-9]{1,2}:)?([0-9]{1,2}):([0-9]{1,2})(.[0-9]{1,3})?\]`

var (
	// Should either be at the beginning of file, or beginning of line
	syncRegex  = regexp.MustCompile(`(^|\n)\s*` + timeRegexString)
	timeRegex  = regexp.MustCompile(timeRegexString)
	lrcIdRegex = regexp.MustCompile(`\[(ar|ti|offset):([^\]]+)\]`)
)

func ToLyrics(language, text string) (*Lyric, error) {
	text = utils.SanitizeText(text)

	lines := strings.Split(text, "\n")

	artist := ""
	title := ""
	var offset *int64 = nil
	structuredLines := []Line{}

	synced := syncRegex.MatchString(text)
	priorLine := ""
	validLine := false
	timestamps := []int64{}

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
					artist = utils.SanitizeText(strings.TrimSpace(idTag[2]))
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
					title = utils.SanitizeText(strings.TrimSpace(idTag[2]))
				}

				continue
			}

			times := timeRegex.FindAllStringSubmatchIndex(line, -1)
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
				timestamps = []int64{}
			}

			end := 0

			// [fullStart, fullEnd, hourStart, hourEnd, minStart, minEnd, secStart, secEnd, msStart, msEnd]
			for _, match := range times {
				var hours, millis int64
				var err error

				// for multiple matches, we need to check that later matches are not
				// in the middle of the string
				if end != 0 {
					middle := strings.TrimSpace(line[end:match[0]])
					if middle != "" {
						break
					}
				}

				end = match[1]

				hourStart := match[2]
				if hourStart != -1 {
					// subtract 1 because group has : at the end
					hourEnd := match[3] - 1
					hours, err = strconv.ParseInt(line[hourStart:hourEnd], 10, 64)
					if err != nil {
						return nil, err
					}
				}

				min, err := strconv.ParseInt(line[match[4]:match[5]], 10, 64)
				if err != nil {
					return nil, err
				}

				sec, err := strconv.ParseInt(line[match[6]:match[7]], 10, 64)
				if err != nil {
					return nil, err
				}

				secStart := match[8]
				if secStart != -1 {
					secEnd := match[9]
					// +1 offset since this capture group contains .
					millis, err = strconv.ParseInt(line[secStart+1:secEnd], 10, 64)
					if err != nil {
						return nil, err
					}

					if secEnd-secStart == 3 {
						millis *= 10
					}
				}

				timeInMillis := (((((hours * 60) + min) * 60) + sec) * 1000) + millis
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

	lyric := Lyric{
		DisplayArtist: artist,
		DisplayTitle:  title,
		Lang:          language,
		Line:          structuredLines,
		Offset:        offset,
		Synced:        synced,
	}

	return &lyric, nil
}

type Lyrics []Lyric
