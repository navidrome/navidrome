package artwork

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"sync"
	"time"

	"github.com/gen2brain/webp"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	xdraw "golang.org/x/image/draw"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

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

	resized, origSize, err := resizeImage(orig, a.size, a.square)
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
	return io.NopCloser(resized), fmt.Sprintf("%s@%d", a.artID, a.size), nil
}

func resizeImage(reader io.Reader, size int, square bool) (io.Reader, int, error) {
	original, _, err := image.Decode(reader)
	if err != nil {
		return nil, 0, err
	}

	bounds := original.Bounds()
	originalSize := max(bounds.Max.X, bounds.Max.Y)

	// Clamp size to original dimensions - upscaling wastes resources and adds no information
	if size > originalSize {
		size = originalSize
	}

	if originalSize <= size && !square {
		return nil, originalSize, nil
	}

	// Calculate aspect-fit dimensions
	srcW, srcH := bounds.Dx(), bounds.Dy()
	scale := float64(size) / float64(max(srcW, srcH))
	dstW := int(float64(srcW) * scale)
	dstH := int(float64(srcH) * scale)

	var dst *image.NRGBA
	var dstRect image.Rectangle
	if square {
		// Square canvas with image centered (transparent padding via zero-initialized NRGBA)
		dst = image.NewNRGBA(image.Rect(0, 0, size, size))
		offsetX := (size - dstW) / 2
		offsetY := (size - dstH) / 2
		dstRect = image.Rect(offsetX, offsetY, offsetX+dstW, offsetY+dstH)
	} else {
		// Tight-fit canvas
		dst = image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
		dstRect = dst.Bounds()
	}
	xdraw.BiLinear.Scale(dst, dstRect, original, bounds, draw.Src, nil)

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if conf.Server.DevJpegCoverArt {
		if square {
			err = png.Encode(buf, dst)
		} else {
			err = jpeg.Encode(buf, dst, &jpeg.Options{Quality: conf.Server.CoverArtQuality})
		}
	} else {
		err = webp.Encode(buf, dst, webp.Options{Quality: conf.Server.CoverArtQuality})
	}
	if err != nil {
		bufPool.Put(buf)
		return nil, originalSize, err
	}
	// Copy bytes before returning buffer to pool (pool may reuse the buffer)
	encoded := make([]byte, buf.Len())
	copy(encoded, buf.Bytes())
	bufPool.Put(buf)
	return bytes.NewReader(encoded), originalSize, nil
}
