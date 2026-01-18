package taglib

import (
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/metadata"
)

type extractor struct {
	baseDir string
}

func (e extractor) Parse(files ...string) (map[string]metadata.Info, error) {
	results := make(map[string]metadata.Info)
	for _, path := range files {
		props, err := e.extractMetadata(path)
		if err != nil {
			continue
		}
		results[path] = *props
	}
	return results, nil
}

func (e extractor) Version() string {
	return Version()
}

func (e extractor) extractMetadata(filePath string) (*metadata.Info, error) {
	fullPath := filepath.Join(e.baseDir, filePath)
	tags, err := Read(fullPath)
	if err != nil {
		log.Warn("extractor: Error reading metadata from file. Skipping", "filePath", fullPath, err)
		return nil, err
	}

	// Parse audio properties
	ap := metadata.AudioProperties{}
	ap.BitRate = parseProp(tags, "__bitrate")
	ap.Channels = parseProp(tags, "__channels")
	ap.SampleRate = parseProp(tags, "__samplerate")
	ap.BitDepth = parseProp(tags, "__bitspersample")
	length := parseProp(tags, "__lengthinmilliseconds")
	ap.Duration = (time.Millisecond * time.Duration(length)).Round(time.Millisecond * 10)

	// Extract basic tags
	parseBasicTag(tags, "__title", "title")
	parseBasicTag(tags, "__artist", "artist")
	parseBasicTag(tags, "__album", "album")
	parseBasicTag(tags, "__comment", "comment")
	parseBasicTag(tags, "__genre", "genre")
	parseBasicTag(tags, "__year", "year")
	parseBasicTag(tags, "__track", "tracknumber")

	// Parse track/disc totals
	parseTuple := func(prop string) {
		tagName := prop + "number"
		tagTotal := prop + "total"
		if value, ok := tags[tagName]; ok && len(value) > 0 {
			parts := strings.Split(value[0], "/")
			tags[tagName] = []string{parts[0]}
			if len(parts) == 2 {
				tags[tagTotal] = []string{parts[1]}
			}
		}
	}
	parseTuple("track")
	parseTuple("disc")

	// Adjust some ID3 tags
	parseLyrics(tags)
	parseTIPL(tags)
	delete(tags, "tmcl") // TMCL is already parsed by TagLib

	return &metadata.Info{
		Tags:            tags,
		AudioProperties: ap,
		HasPicture:      tags["has_picture"] != nil && len(tags["has_picture"]) > 0 && tags["has_picture"][0] == "true",
	}, nil
}

// parseLyrics make sure lyrics tags have language
func parseLyrics(tags map[string][]string) {
	lyrics := tags["lyrics"]
	if len(lyrics) > 0 {
		tags["lyrics:xxx"] = lyrics
		delete(tags, "lyrics")
	}
}

// These are the only roles we support, based on Picard's tag map:
// https://picard-docs.musicbrainz.org/downloads/MusicBrainz_Picard_Tag_Map.html
var tiplMapping = map[string]string{
	"arranger": "arranger",
	"engineer": "engineer",
	"producer": "producer",
	"mix":      "mixer",
	"DJ-mix":   "djmixer",
}

// parseProp parses a property from the tags map and sets it to the target integer.
// It also deletes the property from the tags map after parsing.
func parseProp(tags map[string][]string, prop string) int {
	if value, ok := tags[prop]; ok && len(value) > 0 {
		v, _ := strconv.Atoi(value[0])
		delete(tags, prop)
		return v
	}
	return 0
}

// parseBasicTag checks if a basic tag (like __title, __artist, etc.) exists in the tags map.
// If it does, it moves the value to a more appropriate tag name (like title, artist, etc.),
// and deletes the basic tag from the map. If the target tag already exists, it ignores the basic tag.
func parseBasicTag(tags map[string][]string, basicName string, tagName string) {
	basicValue := tags[basicName]
	if len(basicValue) == 0 {
		return
	}
	delete(tags, basicName)
	if len(tags[tagName]) == 0 {
		tags[tagName] = basicValue
	}
}

// parseTIPL parses the ID3v2.4 TIPL frame string, which is received from TagLib in the format:
//
//	"arranger Andrew Powell engineer Chris Blair engineer Pat Stapley producer Eric Woolfson".
//
// and breaks it down into a map of roles and names, e.g.:
//
//	{"arranger": ["Andrew Powell"], "engineer": ["Chris Blair", "Pat Stapley"], "producer": ["Eric Woolfson"]}.
func parseTIPL(tags map[string][]string) {
	tipl := tags["tipl"]
	if len(tipl) == 0 {
		return
	}

	addRole := func(currentRole string, currentValue []string) {
		if currentRole != "" && len(currentValue) > 0 {
			role := tiplMapping[currentRole]
			tags[role] = append(tags[role], strings.Join(currentValue, " "))
		}
	}

	var currentRole string
	var currentValue []string
	for _, part := range strings.Split(tipl[0], " ") {
		if _, ok := tiplMapping[part]; ok {
			addRole(currentRole, currentValue)
			currentRole = part
			currentValue = nil
			continue
		}
		currentValue = append(currentValue, part)
	}
	addRole(currentRole, currentValue)
	delete(tags, "tipl")
}

var _ local.Extractor = (*extractor)(nil)

func init() {
	local.RegisterExtractor("legacy-taglib", func(_ fs.FS, baseDir string) local.Extractor {
		// ignores fs, as taglib extractor only works with local files
		return &extractor{baseDir}
	})
	conf.AddHook(func() {
		log.Debug("TagLib version", "version", Version())
	})
}
