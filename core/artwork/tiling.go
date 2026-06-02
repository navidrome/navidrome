package artwork

import (
	"bytes"
	"context"
	"image"
	"image/draw"
	"image/png"
	"io"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	xdraw "golang.org/x/image/draw"
)

const tileSize = 600

func toAlbumArtworkIDs(albumIDs []string) []model.ArtworkID {
	return slice.Map(albumIDs, func(id string) model.ArtworkID {
		al := model.Album{ID: id}
		return al.CoverArtID()
	})
}

func createTile(_ context.Context, r io.ReadCloser) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return fillCenter(img, tileSize/2, tileSize/2), nil
}

func createTiledImage(_ context.Context, tiles []image.Image) (io.ReadCloser, error) {
	buf := new(bytes.Buffer)
	var rgba draw.Image
	var err error
	if len(tiles) == 4 {
		rgba = image.NewRGBA(image.Rectangle{Max: image.Point{X: tileSize - 1, Y: tileSize - 1}})
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
