package taglib

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/scanner/metadata"
)

const ExtractorID = "taglib"

type Extractor struct{}

func (e *Extractor) Parse(paths ...string) (map[string]metadata.ParsedTags, error) {
	fileTags := map[string]metadata.ParsedTags{}
	for _, path := range paths {
		tags, err := e.extractMetadata(path)
		if !errors.Is(err, os.ErrPermission) {
			fileTags[path] = tags
		}
	}
	return fileTags, nil
}

func (e *Extractor) CustomMappings() metadata.ParsedTags {
	return metadata.ParsedTags{
		"title":       {"titlesort"},
		"album":       {"albumsort"},
		"artist":      {"artistsort"},
		"tracknumber": {"trck", "_track"},
	}
}

func (e *Extractor) extractMetadata(filePath string) (metadata.ParsedTags, error) {
	tags, err := Read(filePath)
	if err != nil {
		log.Warn("TagLib: Error reading metadata from file. Skipping", "filePath", filePath, err)
		return nil, err
	}

	if length, ok := tags["lengthinmilliseconds"]; ok && len(length) > 0 {
		millis, _ := strconv.Atoi(length[0])
		if duration := float64(millis) / 1000.0; duration > 0 {
			tags["duration"] = []string{strconv.FormatFloat(duration, 'f', 2, 32)}
		}
	}
	// Adjust some ID3 tags
	parseTIPL(tags)
	delete(tags, "tmcl") // TMCL is already parsed by TagLib

	return tags, nil
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

// parseTIPL parses the ID3v2.4 TIPL frame string, which is received from TagLib in the format
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

func init() {
	metadata.RegisterExtractor(ExtractorID, &Extractor{})
}
