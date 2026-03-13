package artwork

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	xdraw "golang.org/x/image/draw"
)

type playlistArtworkReader struct {
	cacheKey
	a  *artwork
	pl model.Playlist
}

const tileSize = 600

func newPlaylistArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*playlistArtworkReader, error) {
	pl, err := artwork.ds.Playlist(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	a := &playlistArtworkReader{
		a:  artwork,
		pl: *pl,
	}
	a.cacheKey.artID = artID
	a.cacheKey.lastUpdate = pl.UpdatedAt

	// Check sidecar and ExternalImageURL local file ModTimes for cache invalidation.
	// If either is newer than the playlist's UpdatedAt, use that instead so the
	// cache is busted when a user replaces a sidecar image or local file reference.
	for _, path := range []string{
		findPlaylistSidecarPath(ctx, pl.Path),
		pl.ExternalImageURL,
	} {
		if path == "" || strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			continue
		}
		if info, err := os.Stat(path); err == nil {
			if info.ModTime().After(a.cacheKey.lastUpdate) {
				a.cacheKey.lastUpdate = info.ModTime()
			}
		}
	}

	return a, nil
}

func (a *playlistArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *playlistArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	return selectImageReader(ctx, a.artID,
		a.fromPlaylistUploadedImage(),
		a.fromPlaylistSidecar(ctx),
		a.fromPlaylistExternalImage(ctx),
		a.fromGeneratedTiledCover(ctx),
		fromAlbumPlaceholder(),
	)
}

func (a *playlistArtworkReader) fromPlaylistUploadedImage() sourceFunc {
	return fromLocalFile(a.pl.UploadedImagePath())
}

func (a *playlistArtworkReader) fromPlaylistSidecar(ctx context.Context) sourceFunc {
	return fromLocalFile(findPlaylistSidecarPath(ctx, a.pl.Path))
}

func (a *playlistArtworkReader) fromPlaylistExternalImage(ctx context.Context) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		imgURL := a.pl.ExternalImageURL
		if imgURL == "" {
			return nil, "", nil
		}
		parsed, err := url.Parse(imgURL)
		if err != nil {
			return nil, "", err
		}
		if parsed.Scheme == "http" || parsed.Scheme == "https" {
			if !conf.Server.EnableM3UExternalAlbumArt {
				return nil, "", nil
			}
			return fromURL(ctx, parsed)
		}
		return fromLocalFile(imgURL)()
	}
}

// fromLocalFile returns a sourceFunc that opens the given local path.
// Returns (nil, "", nil) if path is empty — signalling "not found, try next source".
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

func (a *playlistArtworkReader) fromGeneratedTiledCover(ctx context.Context) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		tiles, err := a.loadTiles(ctx)
		if err != nil {
			return nil, "", err
		}
		r, err := a.createTiledImage(ctx, tiles)
		return r, "", err
	}
}

func toAlbumArtworkIDs(albumIDs []string) []model.ArtworkID {
	return slice.Map(albumIDs, func(id string) model.ArtworkID {
		al := model.Album{ID: id}
		return al.CoverArtID()
	})
}

func (a *playlistArtworkReader) loadTiles(ctx context.Context) ([]image.Image, error) {
	tracksRepo := a.a.ds.Playlist(ctx).Tracks(a.pl.ID, false)
	albumIds, err := tracksRepo.GetAlbumIDs(model.QueryOptions{Max: 4, Sort: "random()"})
	if err != nil {
		log.Error(ctx, "Error getting album IDs for playlist", "id", a.pl.ID, "name", a.pl.Name, err)
		return nil, err
	}
	ids := toAlbumArtworkIDs(albumIds)

	var tiles []image.Image
	for _, id := range ids {
		r, _, err := fromAlbum(ctx, a.a, id)()
		if err == nil {
			tile, err := a.createTile(ctx, r)
			if err == nil {
				tiles = append(tiles, tile)
			}
			_ = r.Close()
		}
		if len(tiles) == 4 {
			break
		}
	}
	switch len(tiles) {
	case 0:
		return nil, errors.New("could not find any eligible cover")
	case 2:
		tiles = append(tiles, tiles[1], tiles[0])
	case 3:
		tiles = append(tiles, tiles[0])
	}
	return tiles, nil
}

func (a *playlistArtworkReader) createTile(_ context.Context, r io.ReadCloser) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return fillCenter(img, tileSize/2, tileSize/2), nil
}

func (a *playlistArtworkReader) createTiledImage(_ context.Context, tiles []image.Image) (io.ReadCloser, error) {
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
	xdraw.BiLinear.Scale(dst, dst.Bounds(), src, cropRect, draw.Src, nil)
	return dst
}
