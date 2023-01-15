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

type emptyIDReader struct {
	artID model.ArtworkID
}

func newEmptyIDReader(_ context.Context, artID model.ArtworkID) (*emptyIDReader, error) {
	a := &emptyIDReader{
		artID: artID,
	}
	return a, nil
}

func (a *emptyIDReader) LastUpdated() time.Time {
	return consts.ServerStart // Invalidate cached placeholder every server start
}

func (a *emptyIDReader) Key() string {
	return fmt.Sprintf("placeholder.%d.0.%d", a.LastUpdated().UnixMilli(), conf.Server.CoverJpegQuality)
}

func (a *emptyIDReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	return selectImageReader(ctx, a.artID, fromAlbumPlaceholder())
}
