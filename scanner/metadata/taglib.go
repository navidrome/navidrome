package metadata

import (
	"os"

	"github.com/dhowden/tag"
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
	} else {
		if hasEmbeddedImage(filePath) {
			parsedTags["has_picture"] = []string{"true"}
		}
	}

	tags := NewTag(filePath, parsedTags, map[string][]string{
		"title":    {"_track", "titlesort"},
		"album":    {"_album", "albumsort"},
		"artist":   {"_artist", "artistsort"},
		"genre":    {"_genre"},
		"date":     {"_year"},
		"track":    {"_track"},
		"duration": {"length"},
	})

	return tags, nil
}

func hasEmbeddedImage(path string) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic while checking for images. Please report this error with a copy of the file", "path", path, r)
		}
	}()
	f, err := os.Open(path)
	if err != nil {
		log.Warn("Error opening file", "filePath", path, err)
		return false
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		log.Warn("Error reading picture tag from file", "filePath", path, err)
		return false
	}

	return m.Picture() != nil
}
