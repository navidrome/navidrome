package engine

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/deluan/gosonic/domain"
	"github.com/dhowden/tag"
)

type Cover interface {
	GetCover(id string, size int, out io.Writer) error
}

type cover struct {
	mfileRepo domain.MediaFileRepository
}

func NewCover(mr domain.MediaFileRepository) Cover {
	return cover{mr}
}

func (c cover) GetCover(id string, size int, out io.Writer) error {
	mf, err := c.mfileRepo.Get(id)
	if err != nil {
		return err
	}

	var img []byte

	if mf != nil && mf.HasCoverArt {
		img, err = readFromTag(mf.Path)
	} else {
		img, err = ioutil.ReadFile("static/default_cover.jpg")
	}

	if err != nil {
		return DataNotFound
	}

	_, err = out.Write(img)
	return err
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

	return m.Picture().Data, nil
}
