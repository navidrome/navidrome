package metadata

import (
	"errors"
	"os"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/scanner/metadata/taglib"
	"github.com/dhowden/tag"
)

type taglibMetadata struct {
	baseMetadata
	hasPicture bool
}

func (m *taglibMetadata) Duration() float32 { return m.parseFloat("length") }
func (m *taglibMetadata) BitRate() int      { return m.parseInt("bitrate") }
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
	md.tags, err = taglib.Read(filePath)
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
		log.Warn("Error reading picture tag from file", "filePath", path, err)
		return false
	}

	return m.Picture() != nil
}

var _ Metadata = (*taglibMetadata)(nil)
var _ Extractor = (*taglibExtractor)(nil)
