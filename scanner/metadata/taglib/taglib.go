package taglib

import (
	"errors"
	"os"
	"strconv"

	"github.com/navidrome/navidrome/log"
)

type Parser struct{}

type parsedTags = map[string][]string

func (e *Parser) Parse(paths ...string) (map[string]parsedTags, error) {
	fileTags := map[string]parsedTags{}
	for _, path := range paths {
		tags, err := e.extractMetadata(path)
		if !errors.Is(err, os.ErrPermission) {
			fileTags[path] = tags
		}
	}
	return fileTags, nil
}

func (e *Parser) extractMetadata(filePath string) (parsedTags, error) {
	tags, err := Read(filePath)
	if err != nil {
		log.Warn("TagLib: Error reading metadata from file. Skipping", "filePath", filePath, err)
		return nil, err
	}

	alternativeTags := map[string][]string{
		"title":       {"titlesort"},
		"album":       {"albumsort"},
		"artist":      {"artistsort"},
		"tracknumber": {"trck", "_track"},
	}

	if length, ok := tags["lengthinmilliseconds"]; ok && len(length) > 0 {
		millis, _ := strconv.Atoi(length[0])
		if duration := float64(millis) / 1000.0; duration > 0 {
			tags["duration"] = []string{strconv.FormatFloat(duration, 'f', 2, 32)}
		}
	}

	for tagName, alternatives := range alternativeTags {
		for _, altName := range alternatives {
			if altValue, ok := tags[altName]; ok {
				tags[tagName] = append(tags[tagName], altValue...)
			}
		}
	}
	return tags, nil
}
