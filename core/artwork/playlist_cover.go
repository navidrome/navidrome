package artwork

import (
	"context"
	"image"
	"image/draw"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	xdraw "golang.org/x/image/draw"
)

const tileSize = 600

func fromLocalFile(path string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		if path == "" {
			return nil, "", nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}
		return f, path, nil
	}
}

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
