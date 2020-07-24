package core

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
)

type Cover interface {
	Get(ctx context.Context, id string, size int, out io.Writer) error
}

func NewCover(ds model.DataStore, cache *FileCache) Cover {
	return &cover{ds: ds, cache: cache}
}

type cover struct {
	ds    model.DataStore
	cache *FileCache
}

type coverInfo struct {
	c          *cover
	path       string
	size       int
	lastUpdate time.Time
}

func (ci *coverInfo) String() string {
	return fmt.Sprintf("%s.%d.%s.%d", ci.path, ci.size, ci.lastUpdate.Format(time.RFC3339Nano), conf.Server.CoverJpegQuality)
}

func (c *cover) Get(ctx context.Context, id string, size int, out io.Writer) error {
	path, lastUpdate, err := c.getCoverPath(ctx, id)
	if err != nil && err != model.ErrNotFound {
		return err
	}

	info := &coverInfo{
		c:          c,
		path:       path,
		size:       size,
		lastUpdate: lastUpdate,
	}

	r, err := c.cache.Get(ctx, info)
	if err != nil {
		log.Error(ctx, "Error accessing image cache", "path", path, "size", size, err)
		return err
	}

	_, err = io.Copy(out, r)
	return err
}

func (c *cover) getCoverPath(ctx context.Context, id string) (path string, lastUpdated time.Time, err error) {
	// If id is an album cover ID
	if strings.HasPrefix(id, "al-") {
		log.Trace(ctx, "Looking for album art", "id", id)
		id = strings.TrimPrefix(id, "al-")
		var al *model.Album
		al, err = c.ds.Album(ctx).Get(id)
		if err != nil {
			return
		}
		if al.CoverArtId == "" {
			err = model.ErrNotFound
		}
		return al.CoverArtPath, al.UpdatedAt, err
	}

	log.Trace(ctx, "Looking for media file art", "id", id)
	// if id is a mediafile cover id
	var mf *model.MediaFile
	mf, err = c.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return
	}
	if mf.HasCoverArt {
		return mf.Path, mf.UpdatedAt, nil
	}

	// if the mediafile does not have a coverArt, fallback to the album cover
	log.Trace(ctx, "Media file does not contain art. Falling back to album art", "id", id, "albumId", "al-"+mf.AlbumID)
	return c.getCoverPath(ctx, "al-"+mf.AlbumID)
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
	err = jpeg.Encode(buf, m, &jpeg.Options{Quality: conf.Server.CoverJpegQuality})
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

func NewImageCache() (*FileCache, error) {
	return NewFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems,
		func(ctx context.Context, arg fmt.Stringer) (io.Reader, error) {
			info := arg.(*coverInfo)
			reader, err := info.c.getCover(ctx, info.path, info.size)
			if err != nil {
				log.Error(ctx, "Error loading cover art", "path", info.path, "size", info.size, err)
				return nil, err
			}
			return reader, nil
		})
}
