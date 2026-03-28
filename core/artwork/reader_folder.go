package artwork

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
)

type folderArtworkReader struct {
	cacheKey
	folder   model.Folder
	imgFiles []string
}

func newFolderArtworkReader(ctx context.Context, a *artwork, artID model.ArtworkID) (*folderArtworkReader, error) {
	folder, err := a.ds.Folder(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}

	absPath := folder.AbsolutePath()
	imgFiles := make([]string, len(folder.ImageFiles))
	for i, img := range folder.ImageFiles {
		imgFiles[i] = filepath.Join(absPath, img)
	}

	r := &folderArtworkReader{
		folder:   *folder,
		imgFiles: imgFiles,
	}
	r.cacheKey.artID = artID
	r.cacheKey.lastUpdate = folder.ImagesUpdatedAt
	return r, nil
}

func (f *folderArtworkReader) Key() string {
	hash := md5.Sum([]byte(conf.Server.CoverArtPriority))
	return fmt.Sprintf("%s.%x", f.cacheKey.Key(), hash)
}

func (f *folderArtworkReader) LastUpdated() time.Time {
	return f.folder.ImagesUpdatedAt
}

func (f *folderArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	var ff []sourceFunc
	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.CoverArtPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded" || pattern == "external":
			// Folders have no embedded tags and no external artwork sources
		case len(f.imgFiles) > 0:
			ff = append(ff, fromExternalFile(ctx, f.imgFiles, pattern))
		}
	}
	return selectImageReader(ctx, f.cacheKey.artID, ff...)
}
