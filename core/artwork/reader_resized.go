package artwork

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type resizedArtworkReader struct {
	artID      model.ArtworkID
	cacheKey   string
	lastUpdate time.Time
	size       int
	square     bool
	a          *artwork
}

func resizedFromOriginal(ctx context.Context, a *artwork, artID model.ArtworkID, size int, square bool) (*resizedArtworkReader, error) {
	r := &resizedArtworkReader{a: a}
	r.artID = artID
	r.size = size
	r.square = square

	// Get lastUpdated and cacheKey from original artwork
	original, err := a.getArtworkReader(ctx, artID, 0, false)
	if err != nil {
		return nil, err
	}
	r.cacheKey = original.Key()
	r.lastUpdate = original.LastUpdated()
	return r, nil
}

func (a *resizedArtworkReader) Key() string {
	baseKey := fmt.Sprintf("%s.%d", a.cacheKey, a.size)
	if a.square {
		return baseKey + ".square"
	}
	return fmt.Sprintf("%s.%d", baseKey, conf.Server.CoverArtQuality)
}

func (a *resizedArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *resizedArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	// Get artwork in original size, possibly from cache
	orig, _, err := a.a.Get(ctx, a.artID, 0, false)
	if err != nil {
		return nil, "", err
	}
	defer orig.Close()

	resized, origSize, err := a.resizeImage(ctx, orig)
	if resized == nil {
		log.Trace(ctx, "Image smaller than requested size", "artID", a.artID, "original", origSize, "resized", a.size, "square", a.square)
	} else {
		log.Trace(ctx, "Resizing artwork", "artID", a.artID, "original", origSize, "resized", a.size, "square", a.square)
	}
	if err != nil {
		log.Warn(ctx, "Could not resize image. Will return image as is", "artID", a.artID, "size", a.size, "square", a.square, err)
	}
	if err != nil || resized == nil {
		// if we couldn't resize the image, return the original
		orig, _, err = a.a.Get(ctx, a.artID, 0, false)
		return orig, "", err
	}
	// Preserve ReadCloser semantics if the resized reader already supports Close
	// (e.g., ffmpeg pipe), otherwise wrap with NopCloser
	if rc, ok := resized.(io.ReadCloser); ok {
		return rc, fmt.Sprintf("%s@%d", a.artID, a.size), nil
	}
	return io.NopCloser(resized), fmt.Sprintf("%s@%d", a.artID, a.size), nil
}

func (a *resizedArtworkReader) resizeImage(ctx context.Context, reader io.Reader) (io.Reader, int, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, fmt.Errorf("reading image data: %w", err)
	}
	return resizeImageData(ctx, a.a.ffmpeg, data, a.size, a.square)
}
