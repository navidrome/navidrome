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
	return fmt.Sprintf("%s.%d", baseKey, conf.Server.CoverJpegQuality)
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

	// Keep a copy of the original data. In case we can't resize it, send it as is
	buf := new(bytes.Buffer)
	r := io.TeeReader(orig, buf)
	defer orig.Close()

	resized, origSize, err := resizeImage(r, a.size, a.square)
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

func resizeImage(reader io.Reader, size int, square bool) (io.Reader, int, error) {
	original, format, err := image.Decode(reader)
	if err != nil {
		return nil, 0, err
	}

	bounds := original.Bounds()
	originalSize := max(bounds.Max.X, bounds.Max.Y)

	if originalSize <= size && !square {
		return nil, originalSize, nil
	}

	var resized image.Image
	if originalSize >= size {
		resized = imaging.Fit(original, size, size, imaging.Lanczos)
	} else {
		if bounds.Max.Y < bounds.Max.X {
			resized = imaging.Resize(original, size, 0, imaging.Lanczos)
		} else {
			resized = imaging.Resize(original, 0, size, imaging.Lanczos)
		}
	}
	if square {
		bg := image.NewRGBA(image.Rect(0, 0, size, size))
		resized = imaging.OverlayCenter(bg, resized, 1)
	}

	buf := new(bytes.Buffer)
	if format == "png" || square {
		err = png.Encode(buf, resized)
	} else {
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: conf.Server.CoverJpegQuality})
	}
	return buf, originalSize, err
}
