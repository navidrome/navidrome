package artwork

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"time"

	"github.com/disintegration/imaging"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type resizedArtworkReader struct {
	artID      model.ArtworkID
	cacheKey   string
	lastUpdate time.Time
	size       int
	a          *artwork
}

func resizedFromOriginal(ctx context.Context, a *artwork, artID model.ArtworkID, size int) (*resizedArtworkReader, error) {
	r := &resizedArtworkReader{a: a}
	r.artID = artID
	r.size = size

	// Get lastUpdated and cacheKey from original artwork
	original, err := a.getArtworkReader(ctx, artID, 0)
	if err != nil {
		return nil, err
	}
	r.cacheKey = original.Key()
	r.lastUpdate = original.LastUpdated()
	return r, nil
}

func (a *resizedArtworkReader) Key() string {
	return fmt.Sprintf(
		"%s.%d.%d",
		a.cacheKey,
		a.size,
		conf.Server.CoverJpegQuality,
	)
}

func (a *resizedArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *resizedArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	// Get artwork in original size, possibly from cache
	orig, _, err := a.a.Get(ctx, a.artID, 0)
	if err != nil {
		return nil, "", err
	}

	// Keep a copy of the original data. In case we can't resize it, send it as is
	buf := new(bytes.Buffer)
	r := io.TeeReader(orig, buf)
	defer orig.Close()

	resized, origSize, err := resizeImage(r, a.size)
	if resized == nil {
		log.Trace(ctx, "Image smaller than requested size", "artID", a.artID, "original", origSize, "resized", a.size)
	} else {
		log.Trace(ctx, "Resizing artwork", "artID", a.artID, "original", origSize, "resized", a.size)
	}
	if err != nil {
		log.Warn(ctx, "Could not resize image. Will return image as is", "artID", a.artID, "size", a.size, err)
	}
	if err != nil || resized == nil {
		// Force finish reading any remaining data
		_, _ = io.Copy(io.Discard, r)
		return io.NopCloser(buf), "", nil //nolint:nilerr
	}
	return io.NopCloser(resized), fmt.Sprintf("%s@%d", a.artID, a.size), nil
}

func resizeImage(reader io.Reader, size int) (io.Reader, int, error) {
	original, format, err := image.Decode(reader)
	if err != nil {
		return nil, 0, err
	}

	bounds := original.Bounds()
	originalSize := max(bounds.Max.X, bounds.Max.Y)

	// Don't upscale the image
	if originalSize <= size {
		return nil, originalSize, nil
	}

	resized := imaging.Fit(original, size, size, imaging.Lanczos)

	buf := new(bytes.Buffer)
	if format == "png" {
		err = png.Encode(buf, resized)
	} else {
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: conf.Server.CoverJpegQuality})
	}
	return buf, originalSize, err
}
