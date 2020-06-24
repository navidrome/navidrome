package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/resources"
	"github.com/deluan/navidrome/utils"
	"github.com/dhowden/tag"
	"github.com/disintegration/imaging"
	"github.com/djherbis/fscache"
)

type Cover interface {
	Get(ctx context.Context, id string, size int, out io.Writer) error
}

type ImageCache fscache.Cache

func NewCover(ds model.DataStore, cache ImageCache) Cover {
	return &cover{ds: ds, cache: cache}
}

type cover struct {
	ds    model.DataStore
	cache fscache.Cache
}

func (c *cover) Get(ctx context.Context, id string, size int, out io.Writer) error {
	id = strings.TrimPrefix(id, "al-")
	path, lastUpdate, err := c.getCoverPath(ctx, id)
	if err != nil && err != model.ErrNotFound {
		return err
	}

	// If cache is disabled, just read the coverart directly from file
	if c.cache == nil {
		log.Trace(ctx, "Retrieving cover art from file", "path", path, "size", size, err)
		reader, err := c.getCover(ctx, path, size)
		if err != nil {
			log.Error(ctx, "Error loading cover art", "path", path, "size", size, err)
		} else {
			_, err = io.Copy(out, reader)
		}
		return err
	}

	cacheKey := imageCacheKey(path, size, lastUpdate)
	r, w, err := c.cache.Get(cacheKey)
	if err != nil {
		log.Error(ctx, "Error reading from image cache", "path", path, "size", size, err)
		return err
	}
	defer r.Close()
	if w != nil {
		log.Trace(ctx, "Image cache miss", "path", path, "size", size, "lastUpdate", lastUpdate)
		go func() {
			defer w.Close()
			reader, err := c.getCover(ctx, path, size)
			if err != nil {
				log.Error(ctx, "Error loading cover art", "path", path, "size", size, err)
				return
			}
			if _, err := io.Copy(w, reader); err != nil {
				log.Error(ctx, "Error saving covert art to cache", "path", path, "size", size, err)
			}
		}()
	} else {
		log.Trace(ctx, "Loading image from cache", "path", path, "size", size, "lastUpdate", lastUpdate)
	}

	_, err = io.Copy(out, r)
	return err
}

func (c *cover) getCoverPath(ctx context.Context, id string) (path string, lastUpdated time.Time, err error) {
	var found bool
	if found, err = c.ds.Album(ctx).Exists(id); err != nil {
		return
	}
	var coverPath string
	if found {
		var al *model.Album
		al, err = c.ds.Album(ctx).Get(id)
		if err != nil {
			return
		}
		if al.CoverArtId == "" {
			err = model.ErrNotFound
			return
		}
		id = al.CoverArtId
		coverPath = al.CoverArtPath
	}
	var mf *model.MediaFile
	mf, err = c.ds.MediaFile(ctx).Get(id)
	if err == nil && mf.HasCoverArt {
		return mf.Path, mf.UpdatedAt, nil
	} else if err != nil && coverPath != "" {
		info, err := os.Stat(coverPath)
		if err != nil {
			return "", time.Time{}, model.ErrNotFound
		}
		return coverPath, info.ModTime(), nil
	} else if err != nil {
		return
	}

	return "", time.Time{}, model.ErrNotFound
}

func imageCacheKey(path string, size int, lastUpdate time.Time) string {
	return fmt.Sprintf("%s.%d.%s", path, size, lastUpdate.Format(time.RFC3339Nano))
}

func (c *cover) getCover(ctx context.Context, path string, size int) (reader io.Reader, err error) {
	defer func() {
		if err != nil {
			log.Warn(ctx, "Error extracting image", "path", path, "size", size, err)
			reader, err = resources.AssetFile().Open(consts.PlaceholderAlbumArt)
		}
	}()

	if path == "" {
		return nil, errors.New("empty path given for cover")
	}

	var data []byte
	if utils.IsAudioFile(path) {
		data, err = readFromTag(path)
	} else {
		data, err = readFromFile(path)
	}

	if err != nil {
		return
	} else if size > 0 {
		data, err = resizeImage(bytes.NewReader(data), size)
	}

	// Confirm the image is valid. Costly, but necessary
	_, _, err = image.Decode(bytes.NewReader(data))
	if err == nil {
		reader = bytes.NewReader(data)
	}

	return
}

func resizeImage(reader io.Reader, size int) ([]byte, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}
	m := imaging.Resize(img, size, size, imaging.Lanczos)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, m, &jpeg.Options{Quality: 75})
	return buf.Bytes(), err
}

func readFromTag(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	picture := m.Picture()
	if picture == nil {
		return nil, errors.New("file does not contain embedded art")
	}
	return picture.Data, nil
}

func readFromFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(f); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func NewImageCache() (ImageCache, error) {
	return newFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems)
}
