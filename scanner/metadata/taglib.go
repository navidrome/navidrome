package metadata

import (
	"errors"
	"os"

	"github.com/deluan/navidrome/log"
	"github.com/dhowden/tag"
	"github.com/nicksellen/audiotags"
)

type taglibMetadata struct {
	baseMetadata
	props      *audiotags.AudioProperties
	hasPicture bool
}

func (m *taglibMetadata) Duration() float32 { return float32(m.props.Length) }
func (m *taglibMetadata) BitRate() int      { return m.props.Bitrate }
func (m *taglibMetadata) HasPicture() bool  { return m.hasPicture }

type taglibExtractor struct{}

func (e *taglibExtractor) Extract(paths ...string) (map[string]Metadata, error) {
	mds := map[string]Metadata{}
	var err error
	for _, path := range paths {
		md, err := e.extractMetadata(path)
		if err == nil {
			mds[path] = md
		}
	}
	return mds, err
}

func (e *taglibExtractor) extractMetadata(filePath string) (*taglibMetadata, error) {
	var err error
	md := &taglibMetadata{}
	md.filePath = filePath
	md.fileInfo, err = os.Stat(filePath)
	if err != nil {
		log.Warn("Error stating file. Skipping", "filePath", filePath, err)
		return nil, errors.New("error stating file")
	}
	md.tags, md.props, err = audiotags.Read(filePath)
	if err != nil {
		log.Warn("Error reading metadata from file. Skipping", "filePath", filePath, err)
		return nil, errors.New("error reading tags")
	}
	md.hasPicture = hasEmbeddedImage(filePath)
	return md, nil
}

func hasEmbeddedImage(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		log.Warn("Error opening file", "filePath", path, err)
		return false
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		log.Warn("Error reading tags from file", "filePath", path, err)
		return false
	}

	return m.Picture() != nil
}

var _ Metadata = (*taglibMetadata)(nil)
var _ Extractor = (*taglibExtractor)(nil)
