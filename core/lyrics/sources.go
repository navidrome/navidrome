package lyrics

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/model"
)

func fromEmbedded(mf *model.MediaFile) (model.LyricList, error) {
	if mf.Lyrics != "" {
		return mf.StructuredLyrics()
	}

	return nil, nil
}

func fromExternalFile(mf *model.MediaFile, suffix string) (model.LyricList, error) {
	basePath := mf.AbsolutePath()
	ext := filepath.Ext(basePath)

	externalLyric := basePath[0:len(basePath)-len(ext)] + suffix

	contents, err := os.ReadFile(externalLyric)

	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	lyrics, err := model.ToLyrics("xxx", string(contents))
	if err != nil {
		return nil, err
	} else if lyrics == nil {
		return nil, nil
	}

	return model.LyricList{*lyrics}, nil
}
