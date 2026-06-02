package artwork

import (
	"context"
	"errors"
	"image"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type playlistArtworkReader struct {
	cacheKey
	a  *artwork
	pl model.Playlist
}

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
		r, err := createTiledImage(ctx, tiles)
		return r, "", err
	}
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
			tile, err := createTile(ctx, r)
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
