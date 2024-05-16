package taglib

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/tag"
	"github.com/navidrome/navidrome/scanner/metadata"
)

type extractor struct{}

func (e extractor) Parse(files ...string) (map[string]tag.Properties, error) {
	results := make(map[string]tag.Properties)
	for _, path := range files {
		props, err := e.extractMetadata(path)
		if errors.Is(err, os.ErrPermission) {
			continue
		}
		results[path] = *props
	}
	return results, nil
}

func (e extractor) Version() string {
	return Version()
}

func (e *extractor) extractMetadata(filePath string) (*tag.Properties, error) {
	tags, err := Read(filePath)
	if err != nil {
		log.Warn("extractor: Error reading metadata from file. Skipping", "filePath", filePath, err)
		return nil, err
	}

	// Parse audio properties
	ap := tag.AudioProperties{}
	if length, ok := tags["_lengthinmilliseconds"]; ok && len(length) > 0 {
		millis, _ := strconv.Atoi(length[0])
		if millis > 0 {
			ap.Duration = (time.Millisecond * time.Duration(millis)).Round(time.Millisecond * 10)
		}
		delete(tags, "_lengthinmilliseconds")
	}
	parseProp := func(prop string, target *int) {
		if value, ok := tags[prop]; ok && len(value) > 0 {
			*target, _ = strconv.Atoi(value[0])
			delete(tags, prop)
		}
	}
	parseProp("_bitrate", &ap.BitRate)
	parseProp("_channels", &ap.Channels)
	parseProp("_samplerate", &ap.SampleRate)

	// Adjust some ID3 tags
	parseTIPL(tags)
	delete(tags, "tmcl") // TMCL is already parsed by extractor

	return &tag.Properties{
		Tags:            tags,
		AudioProperties: ap,
		HasPicture:      tags["has_picture"] != nil && len(tags["has_picture"]) > 0 && tags["has_picture"][0] == "true",
	}, nil
}

// These are the only roles we support, based on Picard's tag map:
// https://picard-docs.musicbrainz.org/downloads/MusicBrainz_Picard_Tag_Map.html
var tiplMapping = map[string]string{
	"arranger": "arranger",
	"engineer": "engineer",
	"producer": "producer",
	"mix":      "mixer",
	"dj-mix":   "djmixer",
}

// parseTIPL parses the ID3v2.4 TIPL frame string, which is received from extractor in the format
//
//	"arranger Andrew Powell engineer Chris Blair engineer Pat Stapley producer Eric Woolfson".
//
// and breaks it down into a map of roles and names, e.g.:
//
//	{"arranger": ["Andrew Powell"], "engineer": ["Chris Blair", "Pat Stapley"], "producer": ["Eric Woolfson"]}.
func parseTIPL(tags metadata.ParsedTags) {
	tipl := tags["tipl"]
	if len(tipl) == 0 {
		return
	}

	addRole := func(tags metadata.ParsedTags, currentRole string, currentValue []string) {
		if currentRole != "" && len(currentValue) > 0 {
			role := tiplMapping[currentRole]
			tags[role] = append(tags[currentRole], strings.Join(currentValue, " "))
		}
	}

	var currentRole string
	var currentValue []string
	for _, part := range strings.Split(tipl[0], " ") {
		if _, ok := tiplMapping[part]; ok {
			addRole(tags, currentRole, currentValue)
			currentRole = part
			currentValue = nil
			continue
		}
		currentValue = append(currentValue, part)
	}
	addRole(tags, currentRole, currentValue)
	delete(tags, "tipl")
}

var _ tag.Extractor = (*extractor)(nil)

func init() {
	tag.RegisterExtractor("taglib", &extractor{})
}
