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
	"strings"

	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/static"
	"github.com/dhowden/tag"
	"github.com/nfnt/resize"
)

type Cover interface {
	Get(ctx context.Context, id string, size int, out io.Writer) error
}

type cover struct {
	ds model.DataStore
}

func NewCover(ds model.DataStore) Cover {
	return &cover{ds}
}

func (c *cover) getCoverPath(id string) (string, error) {
	switch {
	case strings.HasPrefix(id, "al-"):
		id = id[3:]
		al, err := c.ds.Album().Get(id)
		if err != nil {
			return "", err
		}
		return al.CoverArtPath, nil
	default:
		mf, err := c.ds.MediaFile().Get(id)
		if err != nil {
			return "", err
		}
		if mf.HasCoverArt {
			return mf.Path, nil
		}
	}
	return "", model.ErrNotFound
}

func (c *cover) Get(ctx context.Context, id string, size int, out io.Writer) error {
	path, err := c.getCoverPath(id)
	if err != nil && err != model.ErrNotFound {
		return err
	}

	var reader io.Reader

	if err != model.ErrNotFound {
		reader, err = readFromTag(path)
	} else {
		f, err := static.AssetFile().Open("default_cover.jpg")
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

	m := resize.Resize(uint(size), 0, img, resize.NearestNeighbor)
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
