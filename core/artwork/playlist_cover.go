package artwork

import (
	"bytes"
	"context"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	xdraw "golang.org/x/image/draw"
)

const tileSize = 600

// findPlaylistSidecarPath scans the directory of the playlist file for a sidecar
// image file with the same base name (case-insensitive). Returns empty string if
// no matching image is found or if plsPath is empty.
func findPlaylistSidecarPath(ctx context.Context, plsPath string) string {
	if plsPath == "" {
		return ""
	}
	dir := filepath.Dir(plsPath)
	base := strings.TrimSuffix(filepath.Base(plsPath), filepath.Ext(plsPath))

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Warn(ctx, "Could not read directory for playlist sidecar", "dir", dir, err)
		return ""
	}
	for _, entry := range entries {
		name := entry.Name()
		nameBase := strings.TrimSuffix(name, filepath.Ext(name))
		if !entry.IsDir() && strings.EqualFold(nameBase, base) && model.IsImageFile(name) {
			return filepath.Join(dir, name)
		}
	}
	return ""
}

func rect(pos int) image.Rectangle {
	r := image.Rectangle{}
	switch pos {
	case 1:
		r.Min.X = tileSize / 2
	case 2:
		r.Min.Y = tileSize / 2
	case 3:
		r.Min.X = tileSize / 2
		r.Min.Y = tileSize / 2
	}
	r.Max.X = r.Min.X + tileSize/2
	r.Max.Y = r.Min.Y + tileSize/2
	return r
}

// fillCenter crops the source image from the center and scales it to fill dstW x dstH exactly,
// equivalent to imaging.Fill with Center anchor.
func fillCenter(src image.Image, dstW, dstH int) image.Image {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	// Calculate crop rectangle (center crop to match destination aspect ratio)
	srcAspect := float64(srcW) / float64(srcH)
	dstAspect := float64(dstW) / float64(dstH)

	var cropRect image.Rectangle
	if srcAspect > dstAspect {
		// Source is wider — crop horizontally
		cropW := int(float64(srcH) * dstAspect)
		cropX := (srcW - cropW) / 2
		cropRect = image.Rect(srcBounds.Min.X+cropX, srcBounds.Min.Y, srcBounds.Min.X+cropX+cropW, srcBounds.Max.Y)
	} else {
		// Source is taller — crop vertically
		cropH := int(float64(srcW) / dstAspect)
		cropY := (srcH - cropH) / 2
		cropRect = image.Rect(srcBounds.Min.X, srcBounds.Min.Y+cropY, srcBounds.Max.X, srcBounds.Min.Y+cropY+cropH)
	}

	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, cropRect, draw.Src, nil)
	return dst
}

// decodeTile and assembleTiles mirror playlistArtworkReader's createTile/
// createTiledImage, reusing the same rect/fillCenter cropping helpers.
// decodeTile runs on every sampled album's resolved bytes before the processor's
// own maxImageBytes/maxImagePixels guards apply, so it enforces them itself too.
func decodeTile(r io.ReadCloser) (image.Image, error) {
	data, err := readCapped(r)
	if err != nil {
		return nil, err
	}
	img, _, err := decodeCapped(data)
	if err != nil {
		return nil, err
	}
	return fillCenter(img, tileSize/2, tileSize/2), nil
}

func assembleTiles(tiles []image.Image) (io.ReadCloser, error) {
	buf := new(bytes.Buffer)
	var err error
	if len(tiles) == 4 {
		rgba := image.NewRGBA(image.Rectangle{Max: image.Point{X: tileSize - 1, Y: tileSize - 1}})
		draw.Draw(rgba, rect(0), tiles[0], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(1), tiles[1], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(2), tiles[2], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(3), tiles[3], image.Point{}, draw.Src)
		err = png.Encode(buf, rgba)
	} else {
		err = png.Encode(buf, tiles[0])
	}
	if err != nil {
		return nil, err
	}
	return io.NopCloser(buf), nil
}
