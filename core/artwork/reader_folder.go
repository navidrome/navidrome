package artwork

import (
	"context"
	"errors"
	"image"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

type folderArtworkReader struct {
	cacheKey
	a      *artwork
	folder model.Folder
	lib    libraryView
}

func newFolderArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*folderArtworkReader, error) {
	f, err := artwork.ds.Folder(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	lib, err := loadLibraryView(ctx, artwork.ds, f.LibraryID)
	if err != nil {
		return nil, err
	}

	a := &folderArtworkReader{
		a:      artwork,
		folder: *f,
		lib:    lib,
	}
	a.cacheKey.artID = artID
	a.cacheKey.lastUpdate = utils.TimeNewest(f.UpdatedAt, f.CreatedAt)
	if !f.ImagesUpdatedAt.IsZero() {
		a.cacheKey.lastUpdate = utils.TimeNewest(a.cacheKey.lastUpdate, f.ImagesUpdatedAt)
	}

	return a, nil
}

func (a *folderArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *folderArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	return selectImageReader(ctx, a.artID,
		a.fromFolderExternalFile(ctx),
		a.fromGeneratedTiledCover(ctx),
		fromAlbumPlaceholder(),
	)
}

func (a *folderArtworkReader) fromFolderExternalFile(ctx context.Context) sourceFunc {
	if len(a.folder.ImageFiles) == 0 {
		return func() (io.ReadCloser, string, error) { return nil, "", nil }
	}
	var imgFiles []string
	rel := strings.TrimPrefix(path.Join(a.folder.Path, a.folder.Name), "/")
	for _, img := range a.folder.ImageFiles {
		imgFiles = append(imgFiles, path.Join(rel, img))
	}
	slices.SortFunc(imgFiles, compareImageFiles)

	return fromExternalFile(ctx, a.lib.FS, imgFiles, "cover,folder,front")
}

func (a *folderArtworkReader) fromGeneratedTiledCover(ctx context.Context) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		tiles, err := a.loadTiles(ctx)
		if err != nil {
			return nil, "", err
		}
		r, err := createTiledImage(ctx, tiles)
		return r, "", err
	}
}

func (a *folderArtworkReader) loadTiles(ctx context.Context) ([]image.Image, error) {
	// Find top 4 albums in this folder hierarchy
	tracks, err := a.a.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: Eq{"folder_id_recursive": a.folder.ID, "media_file.missing": false},
		Max:     100, // Look at enough tracks to find diverse albums
	})
	if err != nil {
		return nil, err
	}

	albumIDMap := make(map[string]bool)
	var albumIDs []string
	for _, t := range tracks {
		if t.AlbumID != "" && !albumIDMap[t.AlbumID] {
			albumIDMap[t.AlbumID] = true
			albumIDs = append(albumIDs, t.AlbumID)
		}
		if len(albumIDs) == 4 {
			break
		}
	}

	if len(albumIDs) == 0 {
		return nil, errors.New("no albums found in folder hierarchy")
	}

	ids := toAlbumArtworkIDs(albumIDs)
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
