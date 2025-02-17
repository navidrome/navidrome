package core

import (
	"path/filepath"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	. "github.com/navidrome/navidrome/utils/gg"
)

type InspectOutput struct {
	File       string              `json:"file"`
	RawTags    map[string][]string `json:"rawTags"`
	MappedTags *model.MediaFile    `json:"mappedTags,omitempty"`
}

func Inspect(filePath string, libraryId int, folderId string) (*InspectOutput, error) {
	path, file := filepath.Split(filePath)

	s, err := storage.For(path)
	if err != nil {
		return nil, err
	}

	fs, err := s.FS()
	if err != nil {
		return nil, err
	}

	tags, err := fs.ReadTags(file)
	if err != nil {
		return nil, err
	}

	md := metadata.New(path, tags[file])
	result := &InspectOutput{
		File:       filePath,
		RawTags:    tags[file].Tags,
		MappedTags: P(md.ToMediaFile(libraryId, folderId)),
	}

	return result, nil
}
