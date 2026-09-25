package artwork

import (
	"context"
	"errors"
	"io"

	"github.com/navidrome/navidrome/model"
)

// CoverArtReader serves the image getCoverArt would return for an item, minus the placeholder
// fallback. It satisfies core.CoverArtReader, which cannot name *Image without an import cycle.
type CoverArtReader struct {
	artwork Artwork
}

func NewCoverArtReader(artwork Artwork) *CoverArtReader {
	return &CoverArtReader{artwork: artwork}
}

// Read returns model.ErrNotFound when the item has no artwork to serve.
func (c *CoverArtReader) Read(ctx context.Context, artID model.ArtworkID, size int, square bool) (io.ReadCloser, error) {
	img, err := c.artwork.Get(ctx, artID, size, square)
	if errors.Is(err, ErrUnavailable) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return img, nil
}
