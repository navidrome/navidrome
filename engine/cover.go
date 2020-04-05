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
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

func (c *cover) getCoverPath(ctx context.Context, id string) (string, error) {
	var found bool
	var err error
	if found, err = c.ds.Album(ctx).Exists(id); err != nil {
		return "", err
	}
	if found {
		al, err := c.ds.Album(ctx).Get(id)
		if err != nil {
			return "", err
		}
		if al.CoverArtId == "" {
			return "", model.ErrNotFound
		}
		return al.CoverArtPath, nil
	}
	mf, err := c.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return "", err
	}
	if mf.HasCoverArt {
		return mf.Path, nil
	}
	return "", model.ErrNotFound
}

func (c *cover) Get(ctx context.Context, id string, size int, out io.Writer) error {
	id = strings.TrimPrefix(id, "al-")
	path, err := c.getCoverPath(ctx, id)
	if err != nil && err != model.ErrNotFound {
		return err
	}

	var reader io.Reader

	if err != model.ErrNotFound {
		reader, err = readFromTag(path)
	} else {
		var f http.File
		f, err = static.AssetFile().Open("default_cover.jpg")
		if err == nil {
			defer f.Close()
			reader = f
		}
	}

	if err != nil {
		return model.ErrNotFound
	}

	if size > 0 {
		return resizeImage(reader, size, out)
	}
	_, err = io.Copy(out, reader)
	return err
}

func resizeImage(reader io.Reader, size int, out io.Writer) error {
	img, _, err := image.Decode(reader)
	if err != nil {
		return err
	}

	m := imaging.Resize(img, size, size, imaging.Lanczos)
	return jpeg.Encode(out, m, &jpeg.Options{Quality: 75})
}

func readFromTag(path string) (io.Reader, error) {
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
		return nil, errors.New("error extracting art from file " + path)
	}
	return bytes.NewReader(picture.Data), nil
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
