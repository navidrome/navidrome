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

	"github.com/gen2brain/webp"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	xdraw "golang.org/x/image/draw"
)

func init() {
	conf.AddHook(func() {
		// gen2brain/webp selects native (purego/libwebp) vs WASM in its own
		// package init() and exposes the result only via webp.Dynamic(); there is
		// no runtime way to switch back. On 32-bit ARM/x86 the purego callback path
		// crashes (issue #5597), so those builds must be compiled with the
		// "nodynamic" tag (see Dockerfile), which makes webp.Dynamic() report an
		// error here and forces the safe WASM path.
		if err := webp.Dynamic(); err != nil {
			log.Debug("Using WASM WebP encoder/decoder", "reason", err)
		} else {
			log.Debug("Using native libwebp for WebP encoding/decoding")
		}
	})
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// resizeImageData resizes raw image bytes to fit size, preserving animation where
// possible. A nil reader means the image was already within bounds (no resize needed).
func resizeImageData(ctx context.Context, ffm ffmpeg.FFmpeg, data []byte, size int, square bool) (io.Reader, int, error) {
	// Preserve animation for animated images
	if isAnimatedGIF(data) {
		if ffm.IsAvailable() {
			// Animated GIF: convert to animated WebP via ffmpeg (with optional resize)
			r, err := ffm.ConvertAnimatedImage(ctx, bytes.NewReader(data), size, conf.Server.CoverArtQuality)
			if err == nil {
				return r, 0, nil
			}
			log.Warn(ctx, "Could not convert animated GIF, falling back to static", err)
		}
	} else if isAnimatedWebP(data) || isAnimatedPNG(data) {
		// Animated WebP/APNG: return original as-is (ffmpeg can't re-encode these)
		return bytes.NewReader(data), 0, nil
	}

	return resizeStaticImage(data, size, square)
}

// toFastScaleType converts images whose concrete type has no optimized scaler
// in x/image/draw (e.g. *image.NYCbCrA from WebP, *image.Paletted from indexed
// PNGs) into *image.RGBA, which has a fast path. Without this, CatmullRom.Scale
// falls back to a generic per-pixel At()/RGBA() loop that is several times
// slower. Fast-path types are returned unchanged.
func toFastScaleType(img image.Image) image.Image {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA, *image.Gray, *image.YCbCr:
		return img
	default:
		rgba := image.NewRGBA(img.Bounds())
		draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
		return rgba
	}
}

func resizeStaticImage(data []byte, size int, square bool) (io.Reader, int, error) {
	original, format, err := image.Decode(bytes.NewReader(data))
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
	original = toFastScaleType(original)
	xdraw.CatmullRom.Scale(dst, dstRect, original, bounds, draw.Src, nil)

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if conf.Server.EnableWebPEncoding {
		err = webp.Encode(buf, dst, webp.Options{Quality: conf.Server.CoverArtQuality})
	} else if format == "png" || square {
		err = png.Encode(buf, dst)
	} else {
		err = jpeg.Encode(buf, dst, &jpeg.Options{Quality: conf.Server.CoverArtQuality})
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

// formatQualityTag folds the encoder config (WebP toggle + quality) into a cache-key
// fragment, so flipping either setting invalidates previously-encoded sized artwork.
func formatQualityTag() string {
	if conf.Server.EnableWebPEncoding {
		return fmt.Sprintf("webp%d", conf.Server.CoverArtQuality)
	}
	return fmt.Sprintf("q%d", conf.Server.CoverArtQuality)
}
