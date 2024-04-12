package artwork

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
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

func asImageReader(r io.Reader) (io.Reader, string, error) {
	br := bufio.NewReader(r)
	buf, err := br.Peek(512)
	if err == io.EOF && len(buf) > 0 {
		// Check if there are enough bytes to detect type
		typ := http.DetectContentType(buf)
		if typ != "" {
			return br, typ, nil
		}
	}
	if err != nil {
		return nil, "", err
	}
	return br, http.DetectContentType(buf), nil
}

func resizeImage(reader io.Reader, size int) (io.Reader, int, error) {
	r, format, err := asImageReader(reader)
	if err != nil {
		return nil, 0, err
	}

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, 0, err
	}

	// Don't upscale the image
	bounds := img.Bounds()
	originalSize := max(bounds.Max.X, bounds.Max.Y)
	if originalSize <= size {
		return nil, originalSize, nil
	}

	var m *image.NRGBA
	// Preserve the aspect ratio of the image.
	if bounds.Max.X > bounds.Max.Y {
		m = imaging.Resize(img, size, 0, imaging.Lanczos)
	} else {
		m = imaging.Resize(img, 0, size, imaging.Lanczos)
	}

	buf := new(bytes.Buffer)
	buf.Reset()
	if format == "image/png" {
		err = png.Encode(buf, m)
	} else {
		err = jpeg.Encode(buf, m, &jpeg.Options{Quality: conf.Server.CoverJpegQuality})
	}
	return buf, originalSize, err
}
