package artwork

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

type artistReader struct {
	cacheKey
	a            *artwork
	em           core.ExternalMetadata
	artist       model.Artist
	artistFolder string
	imgFiles     []string
}

func newArtistReader(ctx context.Context, artwork *artwork, artID model.ArtworkID, em core.ExternalMetadata) (*artistReader, error) {
	ar, err := artwork.ds.Artist(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	// Only consider albums where the artist is the sole album artist.
	als, err := artwork.ds.Album(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Eq{"album_artist_id": artID.ID},
			squirrel.Eq{"json_array_length(participants, '$.albumartist')": 1},
		},
	})
	if err != nil {
		return nil, err
	}
	albumPaths, imgFiles, imagesUpdatedAt, err := loadAlbumFoldersPaths(ctx, artwork.ds, als...)
	if err != nil {
		return nil, err
	}
	artistFolder, artistFolderLastUpdate, err := loadArtistFolder(ctx, artwork.ds, als, albumPaths)
	if err != nil {
		return nil, err
	}
	a := &artistReader{
		a:            artwork,
		em:           em,
		artist:       *ar,
		artistFolder: artistFolder,
		imgFiles:     imgFiles,
	}
	// TODO Find a way to factor in the ExternalUpdateInfoAt in the cache key. Problem is that it can
	// change _after_ retrieving from external sources, making the key invalid
	//a.cacheKey.lastUpdate = ar.ExternalInfoUpdatedAt

	a.cacheKey.lastUpdate = *imagesUpdatedAt
	if artistFolderLastUpdate.After(a.cacheKey.lastUpdate) {
		a.cacheKey.lastUpdate = artistFolderLastUpdate
	}
	a.cacheKey.artID = artID
	return a, nil
}

func (a *artistReader) Key() string {
	hash := md5.Sum([]byte(conf.Server.Agents + conf.Server.Spotify.ID))
	return fmt.Sprintf(
		"%s.%t.%x",
		a.cacheKey.Key(),
		conf.Server.EnableExternalServices,
		hash,
	)
}

func (a *artistReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *artistReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	var ff = a.fromArtistArtPriority(ctx, conf.Server.ArtistArtPriority)
	return selectImageReader(ctx, a.artID, ff...)
}

func (a *artistReader) fromArtistArtPriority(ctx context.Context, priority string) []sourceFunc {
	var ff []sourceFunc
	for _, pattern := range strings.Split(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "external":
			ff = append(ff, fromArtistExternalSource(ctx, a.artist, a.em))
		case strings.HasPrefix(pattern, "album/"):
			ff = append(ff, fromExternalFile(ctx, a.imgFiles, strings.TrimPrefix(pattern, "album/")))
		default:
			ff = append(ff, fromArtistFolder(ctx, a.artistFolder, pattern))
		}
	}
	return ff
}

func fromArtistFolder(ctx context.Context, artistFolder string, pattern string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		fsys := os.DirFS(artistFolder)
		matches, err := fs.Glob(fsys, pattern)
		if err != nil {
			log.Warn(ctx, "Error matching artist image pattern", "pattern", pattern, "folder", artistFolder)
			return nil, "", err
		}
		if len(matches) == 0 {
			return nil, "", fmt.Errorf(`no matches for '%s' in '%s'`, pattern, artistFolder)
		}
		for _, m := range matches {
			filePath := filepath.Join(artistFolder, m)
			if !model.IsImageFile(m) {
				continue
			}
			f, err := os.Open(filePath)
			if err != nil {
				log.Warn(ctx, "Could not open cover art file", "file", filePath, err)
				return nil, "", err
			}
			return f, filePath, nil
		}
		return nil, "", nil
	}
}

func loadArtistFolder(ctx context.Context, ds model.DataStore, albums model.Albums, paths []string) (string, time.Time, error) {
	if len(albums) == 0 {
		return "", time.Time{}, nil
	}
	libID := albums[0].LibraryID // Just need one of the albums, as they should all be in the same Library

	folderPath := str.LongestCommonPrefix(paths)
	if !strings.HasSuffix(folderPath, string(filepath.Separator)) {
		folderPath, _ = filepath.Split(folderPath)
	}
	folderPath = filepath.Dir(folderPath)

	// Manipulate the path to get the folder ID
	// TODO: This is a bit hacky, but it's the easiest way to get the folder ID, ATM
	libPath := core.AbsolutePath(ctx, ds, libID, "")
	folderID := model.FolderID(model.Library{ID: libID, Path: libPath}, folderPath)

	log.Trace(ctx, "Calculating artist folder details", "folderPath", folderPath, "folderID", folderID,
		"libPath", libPath, "libID", libID, "albumPaths", paths)

	// Get the last update time for the folder
	folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"folder.id": folderID, "missing": false}})
	if err != nil || len(folders) == 0 {
		log.Warn(ctx, "Could not find folder for artist", "folderPath", folderPath, "id", folderID,
			"libPath", libPath, "libID", libID, err)
		return "", time.Time{}, err
	}
	return folderPath, folders[0].ImagesUpdatedAt, nil
}
