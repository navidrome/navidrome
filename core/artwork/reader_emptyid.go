package artwork

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
)

type placeholderReader struct {
	artID model.ArtworkID
}

func newPlaceholderReader(_ context.Context, artID model.ArtworkID) (*placeholderReader, error) {
	a := &placeholderReader{
		artID: artID,
	}
	return a, nil
}

func (a *placeholderReader) LastUpdated() time.Time {
	return time.Now() // Basically make it non-cacheable
}

func (a *placeholderReader) Key() string {
	return fmt.Sprintf("0.%d.0.%d", a.LastUpdated().UnixMilli(), conf.Server.CoverJpegQuality)
}

func (a *placeholderReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	r, source := extractImage(ctx, a.artID, fromPlaceholder())
	return r, source, nil
}
