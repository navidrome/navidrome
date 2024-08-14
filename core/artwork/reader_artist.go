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
	"github.com/navidrome/navidrome/consts"
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
	files        string
}

func newArtistReader(ctx context.Context, artwork *artwork, artID model.ArtworkID, em core.ExternalMetadata) (*artistReader, error) {
	ar, err := artwork.ds.Artist(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	als, err := artwork.ds.Album(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_artist_id": artID.ID}})
	if err != nil {
		return nil, err
	}
	a := &artistReader{
		a:      artwork,
		em:     em,
		artist: *ar,
	}
	// TODO Find a way to factor in the ExternalUpdateInfoAt in the cache key. Problem is that it can
	// change _after_ retrieving from external sources, making the key invalid
	//a.cacheKey.lastUpdate = ar.ExternalInfoUpdatedAt
	var files []string
	var paths []string
	for _, al := range als {
		files = append(files, al.ImageFiles)
		paths = append(paths, splitList(al.Paths)...)
		if a.cacheKey.lastUpdate.Before(al.UpdatedAt) {
			a.cacheKey.lastUpdate = al.UpdatedAt
		}
	}
	a.files = strings.Join(files, consts.Zwsp)
	a.artistFolder = str.LongestCommonPrefix(paths)
	if !strings.HasSuffix(a.artistFolder, string(filepath.Separator)) {
		a.artistFolder, _ = filepath.Split(a.artistFolder)
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
			ff = append(ff, fromExternalFile(ctx, a.files, strings.TrimPrefix(pattern, "album/")))
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
