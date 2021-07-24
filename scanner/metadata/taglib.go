package metadata

import (
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/scanner/metadata/taglib"
)

type taglibExtractor struct{}

func (e *taglibExtractor) Extract(paths ...string) (map[string]*Tags, error) {
	fileTags := map[string]*Tags{}
	for _, path := range paths {
		tags, err := e.extractMetadata(path)
		if err == nil {
			fileTags[path] = tags
		}
	}
	return fileTags, nil
}

func (e *taglibExtractor) extractMetadata(filePath string) (*Tags, error) {
	parsedTags, err := taglib.Read(filePath)
	if err != nil {
		log.Warn("Error reading metadata from file. Skipping", "filePath", filePath, err)
	}

	tags := NewTags(filePath, parsedTags, map[string][]string{
		"title":  {"_track", "titlesort"},
		"album":  {"_album", "albumsort"},
		"artist": {"_artist", "artistsort"},
		"date":   {"_year"},
		"track":  {"_track"},
	})

	return tags, nil
}
