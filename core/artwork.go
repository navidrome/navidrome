package core

import (
	"bytes"
	"context"
	"errors"
	_ "image/gif"
	_ "image/png"
	"io"
	"os"

	"github.com/dhowden/tag"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	_ "golang.org/x/image/webp"
)

type Artwork interface {
	Get(ctx context.Context, id string, size int) (io.ReadCloser, error)
}

func NewArtwork(ds model.DataStore) Artwork {
	return &artwork{ds: ds}
}

type artwork struct {
	ds model.DataStore
}

func (a *artwork) Get(ctx context.Context, id string, size int) (io.ReadCloser, error) {
	r, _, err := a.get(ctx, id, size)
	return r, err
}

func (a *artwork) get(ctx context.Context, id string, size int) (io.ReadCloser, string, error) {
	artId, err := model.ParseArtworkID(id)
	if err != nil {
		return nil, "", errors.New("invalid ID")
	}
	id = artId.ID
	al, err := a.ds.Album(ctx).Get(id)
	if errors.Is(err, model.ErrNotFound) {
		r, path := fromPlaceholder()()
		return r, path, nil
	}
	if err != nil {
		return nil, "", err
	}
	r, path := extractImage(ctx, artId,
		fromTag(al.EmbedArtPath),
		fromPlaceholder(),
	)
	return r, path, nil
}

func extractImage(ctx context.Context, artId model.ArtworkID, extractFuncs ...func() (io.ReadCloser, string)) (io.ReadCloser, string) {
	for _, f := range extractFuncs {
		r, path := f()
		if r != nil {
			log.Trace(ctx, "Found artwork", "artId", artId, "path", path)
			return r, path
		}
	}
	log.Error(ctx, "extractImage should never reach this point!", "artId", artId, "path")
	return nil, ""
}

func fromTag(path string) func() (io.ReadCloser, string) {
	return func() (io.ReadCloser, string) {
		f, err := os.Open(path)
		if err != nil {
			return nil, ""
		}
		defer f.Close()

		m, err := tag.ReadFrom(f)
		if err != nil {
			return nil, ""
		}

		picture := m.Picture()
		if picture == nil {
			return nil, ""
		}
		return io.NopCloser(bytes.NewReader(picture.Data)), path
	}
}

func fromPlaceholder() func() (io.ReadCloser, string) {
	return func() (io.ReadCloser, string) {
		r, _ := resources.FS().Open(consts.PlaceholderAlbumArt)
		return r, consts.PlaceholderAlbumArt
	}
}
