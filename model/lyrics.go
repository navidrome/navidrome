package model

import (
	"regexp"
	"strconv"
	"strings"

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
const timeRegexString = `(\[(([0-9]{1,2}):)?([0-9]{1,2}):([0-9]{1,2})(\.([0-9]{1,3}))?\])`

var (
	lineRegex  = regexp.MustCompile(timeRegexString + "([^\n]+)")
	lrcIdRegex = regexp.MustCompile(`\[(ar|ti|offset):([^\]]+)\]`)
)

func ToLyrics(language, text string) (*Lyric, error) {
	text = utils.SanitizeText(text)

	lines := strings.Split(text, "\n")
	synced := true

	artist := ""
	title := ""
	var offset *int64 = nil
	structuredLines := []Line{}

	for _, line := range lines {
		line := strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var text string
		var time *int64 = nil

		if synced {
			idTag := lrcIdRegex.FindStringSubmatch(line)
			if idTag != nil {
				switch idTag[1] {
				case "ar":
					artist = idTag[2]
				case "offset":
					{
						off, err := strconv.ParseInt(idTag[2], 10, 64)
						if err != nil {
							return nil, err
						}

						offset = &off
					}
				case "ti":
					title = idTag[2]
				}

				continue
			}

			syncedMatch := lineRegex.FindStringSubmatch(line)
			if syncedMatch == nil {
				synced = false
				text = line
			} else {
				var hours int64
				var err error

				if syncedMatch[3] != "" {
					hours, err = strconv.ParseInt(syncedMatch[3], 10, 64)
					if err != nil {
						return nil, err
					}
				}

				min, err := strconv.ParseInt(syncedMatch[4], 10, 64)
				if err != nil {
					return nil, err
				}

				sec, err := strconv.ParseInt(syncedMatch[5], 10, 64)
				if err != nil {
					return nil, err
				}

				millis, err := strconv.ParseInt(syncedMatch[7], 10, 64)
				if err != nil {
					return nil, err
				}

				if len(syncedMatch[7]) == 2 {
					millis *= 10
				}

				timeInMillis := (((((hours * 60) + min) * 60) + sec) * 1000) + millis
				time = &timeInMillis
				text = syncedMatch[8]
			}
		} else {
			text = line
		}

		structuredLines = append(structuredLines, Line{
			Start: time,
			Value: text,
		})
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
