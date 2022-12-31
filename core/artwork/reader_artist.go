package artwork

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
)

type artistReader struct {
	artID model.ArtworkID
}

func newArtistReader(_ context.Context, _ *artwork, artID model.ArtworkID) (*artistReader, error) {
	a := &artistReader{
		artID: artID,
	}
	return a, nil
}

func (a *artistReader) LastUpdated() time.Time {
	return consts.ServerStart // Invalidate cached placeholder every server start
}

func (a *artistReader) Key() string {
	return fmt.Sprintf("placeholder.%d.0.%d", a.LastUpdated().UnixMilli(), conf.Server.CoverJpegQuality)
}

func (a *artistReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	return selectImageReader(ctx, a.artID, fromArtistPlaceholder())
}
