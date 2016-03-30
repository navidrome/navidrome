package engine

import (
	"bytes"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"

	"github.com/deluan/gosonic/domain"
	"github.com/dhowden/tag"
	"github.com/nfnt/resize"
)

type Cover interface {
	Get(id string, size int, out io.Writer) error
}

type cover struct {
	mfileRepo domain.MediaFileRepository
	albumRepo domain.AlbumRepository
}

func NewCover(mr domain.MediaFileRepository, alr domain.AlbumRepository) Cover {
	return &cover{mr, alr}
}

func (c *cover) getCoverPath(id string) (string, error) {
	switch {
	case strings.HasPrefix(id, "al-"):
		id = id[3:]
		al, err := c.albumRepo.Get(id)
		if err != nil {
			return "", err
		}
		return al.CoverArtPath, nil
	default:
		mf, err := c.mfileRepo.Get(id)
		if err != nil {
			return "", err
		}
		if mf.HasCoverArt {
			return mf.Path, nil
		}
	}
	return "", domain.ErrNotFound
}

func (c *cover) Get(id string, size int, out io.Writer) error {
	path, err := c.getCoverPath(id)
	if err != nil && err != domain.ErrNotFound {
		return err
	}

	var reader io.Reader

	if err != domain.ErrNotFound {
		reader, err = readFromTag(path)
	} else {
		var f *os.File
		f, err = os.Open("static/default_cover.jpg")
		if err == nil {
			defer f.Close()
			reader = f
		}
	}

	if err != nil {
		return domain.ErrNotFound
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

	return bytes.NewReader(m.Picture().Data), nil
}
