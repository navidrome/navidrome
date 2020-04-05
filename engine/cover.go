package engine

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/static"
	"github.com/dhowden/tag"
	"github.com/disintegration/imaging"
	"github.com/djherbis/fscache"
	"github.com/dustin/go-humanize"
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

func (c *cover) getCoverPath(ctx context.Context, id string) (string, *time.Time, error) {
	var found bool
	var err error
	if found, err = c.ds.Album(ctx).Exists(id); err != nil {
		return "", nil, err
	}
	if found {
		al, err := c.ds.Album(ctx).Get(id)
		if err != nil {
			return "", nil, err
		}
		if al.CoverArtId == "" {
			return "", nil, model.ErrNotFound
		}
		id = al.CoverArtId
	}
	mf, err := c.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return "", nil, err
	}
	if mf.HasCoverArt {
		return mf.Path, &mf.UpdatedAt, nil
	}
	return "", nil, model.ErrNotFound
}

func (c *cover) Get(ctx context.Context, id string, size int, out io.Writer) error {
	id = strings.TrimPrefix(id, "al-")
	path, _, err := c.getCoverPath(ctx, id)
	if err != nil && err != model.ErrNotFound {
		return err
	}

	reader, err := c.getCover(ctx, path, size)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, reader)
	return err
}

func (c *cover) getCover(ctx context.Context, path string, size int) (reader io.Reader, err error) {
	defer func() {
		if err != nil {
			log.Warn(ctx, "Error extracting image", "path", path, "size", size, err)
			reader, err = static.AssetFile().Open("navidrome-310x310.png")
		}
	}()
	var data []byte
	data, err = readFromTag(path)

	if err == nil && size > 0 {
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

func NewImageCache() (ImageCache, error) {
	cacheSize, err := humanize.ParseBytes(conf.Server.ImageCacheSize)
	if err != nil {
		cacheSize = consts.DefaultImageCacheSize
	}
	lru := fscache.NewLRUHaunter(consts.DefaultImageCacheMaxItems, int64(cacheSize), consts.DefaultImageCachePurgeInterval)
	h := fscache.NewLRUHaunterStrategy(lru)
	cacheFolder := filepath.Join(conf.Server.DataFolder, consts.ImageCacheDir)
	log.Info("Creating image cache", "path", cacheFolder, "maxSize", humanize.Bytes(cacheSize),
		"cleanUpInterval", consts.DefaultImageCachePurgeInterval)
	fs, err := fscache.NewFs(cacheFolder, 0755)
	if err != nil {
		return nil, err
	}
	return fscache.NewCacheWithHaunter(fs, h)
}
