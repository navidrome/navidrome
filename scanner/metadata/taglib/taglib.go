package taglib

import (
	"errors"
	"os"
	"strconv"

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

	return tags, nil
}

func init() {
	metadata.RegisterExtractor(ExtractorID, &Extractor{})
}
