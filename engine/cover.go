package engine

import (
	"io"
	"os"

	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"bytes"
	"image/jpeg"

	"github.com/deluan/gosonic/domain"
	"github.com/dhowden/tag"
	"github.com/nfnt/resize"
)

type Cover interface {
	Get(id string, size int, out io.Writer) error
}

type cover struct {
	mfileRepo domain.MediaFileRepository
}

func NewCover(mr domain.MediaFileRepository) Cover {
	return cover{mr}
}

func (c cover) Get(id string, size int, out io.Writer) error {
	mf, err := c.mfileRepo.Get(id)
	if err != nil {
		return err
	}

	var reader io.Reader

	if mf != nil && mf.HasCoverArt {
		reader, err = readFromTag(mf.Path)
	} else {
		f, err := os.Open("static/default_cover.jpg")
		if err == nil {
			defer f.Close()
			reader = f
		}
	}

	if err != nil {
		return DataNotFound
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
