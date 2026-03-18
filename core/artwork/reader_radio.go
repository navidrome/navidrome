package artwork

import (
	"context"
	"io"
	"time"

	"github.com/navidrome/navidrome/model"
)

type radioArtworkReader struct {
	cacheKey
	a     *artwork
	radio model.Radio
}

func newRadioArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*radioArtworkReader, error) {
	r, err := artwork.ds.Radio(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	a := &radioArtworkReader{a: artwork, radio: *r}
	a.cacheKey.artID = artID
	a.cacheKey.lastUpdate = r.UpdatedAt
	return a, nil
}

func (a *radioArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *radioArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	return selectImageReader(ctx, a.artID,
		a.fromRadioUploadedImage(),
	)
}

func (a *radioArtworkReader) fromRadioUploadedImage() sourceFunc {
	return fromLocalFile(a.radio.UploadedImagePath())
}
