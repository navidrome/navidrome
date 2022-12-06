package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/dhowden/tag"
	"github.com/disintegration/imaging"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/cache"
	"golang.org/x/exp/slices"
	_ "golang.org/x/image/webp"
)

type Artwork interface {
	Get(ctx context.Context, id string, size int) (io.ReadCloser, error)
}

type ArtworkCache cache.FileCache

func NewArtwork(ds model.DataStore, cache ArtworkCache) Artwork {
	return &artwork{ds: ds, cache: cache}
}

type artwork struct {
	ds    model.DataStore
	cache cache.FileCache
}

const albumArtworkIdPrefix = "al-"

type artworkKey struct {
	a          *artwork
	artworkId  model.ArtworkID
	size       int
	lastUpdate time.Time
}

func (k *artworkKey) Key() string {
	return fmt.Sprintf("%s.%d.%d.%d", k.artworkId.ID, k.size, k.artworkId.LastAccess.UnixNano(), conf.Server.CoverJpegQuality)
}

func (a *artwork) Get(ctx context.Context, id string, size int) (io.ReadCloser, error) {
	var artworkId model.ArtworkID
	var err error
	if id != "" {
		artworkId, err = model.ParseArtworkID(id)
		if err != nil {
			return nil, err
		}
	}
	key := &artworkKey{a: a, artworkId: artworkId, size: size}

	r, err := a.cache.Get(ctx, key)
	if err != nil {
		log.Error(ctx, "Error accessing image cache", "id", id, "size", size, err)
		return nil, err
	}
	return r, err
}

func (a *artwork) getImagePath(ctx context.Context, id model.ArtworkID) (path string, err error) {
	if id.Kind == model.KindMediaFileArtwork {
		log.Trace(ctx, "Looking for media file art", "id", id)
		// Check if id is a mediaFile id
		var mf *model.MediaFile
		mf, err = a.ds.MediaFile(ctx).Get(id.ID)
		if err != nil {
			return "", err
		}
		id = mf.CoverArtID()
		path = mf.Path
	}
	if id.Kind == model.KindAlbumArtwork {
		return a.getAlbumArtPath(ctx, id.ID)
	}
	return path, nil
}

func (a *artwork) getAlbumArtPath(ctx context.Context, id string) (string, error) {
	mfs, err := a.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": id}})
	if err != nil {
		return "", nil
	}
	var paths []string
	var embeddedPath string
	for _, mf := range mfs {
		dir, _ := filepath.Split(mf.Path)
		paths = append(paths, dir)
		if embeddedPath == "" && mf.HasCoverArt {
			embeddedPath = mf.Path
		}
	}
	if len(paths) == 0 && embeddedPath == "" {
		return "", model.ErrNotFound
	}

	sort.Strings(paths)
	paths = slices.Compact(paths)
	return getAlbumCoverFromPath(paths, embeddedPath), nil
}

// getAlbumCoverFromPath accepts a path to a file, and returns a path to an eligible cover image from the
// file's directory (as configured with CoverArtPriority). If no cover file is found, among
// available choices, or an error occurs, an empty string is returned. If HasEmbeddedCover is true,
// and 'embedded' is matched among eligible choices, GetCoverFromPath will return early with an
// empty path.
func getAlbumCoverFromPath(albumPaths []string, embeddedPath string) string {
	for _, p := range strings.Split(conf.Server.CoverArtPriority, ",") {
		pat := strings.ToLower(strings.TrimSpace(p))
		if pat == "embedded" {
			if embeddedPath != "" {
				return embeddedPath
			}
			continue
		}

		for _, path := range albumPaths {
			glob := filepath.Join(path, p)
			matches, err := filepath.Glob(glob)
			if err != nil {
				log.Warn("Error searching for cover art", "path", glob)
				continue
			}
			if len(matches) > 0 {
				return matches[0]
			}
		}
	}

	return ""
}

func (a *artwork) getArtwork(ctx context.Context, id model.ArtworkID, path string, size int) (reader io.ReadCloser, err error) {
	defer func() {
		if err != nil {
			log.Warn(ctx, "Error extracting image", "path", path, "size", size, err)
			reader, err = resources.FS().Open(consts.PlaceholderAlbumArt)

			if size != 0 && err == nil {
				var r io.ReadCloser
				r, err = resources.FS().Open(consts.PlaceholderAlbumArt)
				reader, err = resizeImage(r, size, true)
			}
		}
	}()

	if path == "" {
		return nil, errors.New("empty path given for artwork")
	}

	if size == 0 {
		// If requested original size, just read from the file
		if utils.IsAudioFile(path) {
			reader, err = readFromTag(path)
		} else {
			reader, err = readFromFile(path)
		}
	} else {
		// If requested a resized image, get the original (possibly from cache) and resize it
		var r io.ReadCloser
		r, err = a.Get(ctx, id.String(), 0)
		if err != nil {
			return
		}
		defer r.Close()
		reader, err = resizeImage(r, size, false)
	}

	return reader, err
}

func resizeImage(reader io.Reader, size int, usePng bool) (io.ReadCloser, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	// Preserve the aspect ratio of the image.
	var m *image.NRGBA
	bounds := img.Bounds()
	if bounds.Max.X > bounds.Max.Y {
		m = imaging.Resize(img, size, 0, imaging.Lanczos)
	} else {
		m = imaging.Resize(img, 0, size, imaging.Lanczos)
	}

	buf := new(bytes.Buffer)
	if usePng {
		err = png.Encode(buf, m)
	} else {
		err = jpeg.Encode(buf, m, &jpeg.Options{Quality: conf.Server.CoverJpegQuality})
	}
	return io.NopCloser(buf), err
}

func readFromTag(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	picture := m.Picture()
	if picture == nil {
		return nil, errors.New("file does not contain embedded art")
	}
	return io.NopCloser(bytes.NewReader(picture.Data)), nil
}

func readFromFile(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

var (
	onceImageCache     sync.Once
	instanceImageCache ArtworkCache
)

func GetImageCache() ArtworkCache {
	onceImageCache.Do(func() {
		instanceImageCache = cache.NewFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems,
			func(ctx context.Context, arg cache.Item) (io.Reader, error) {
				info := arg.(*artworkKey)
				path, err := info.a.getImagePath(ctx, info.artworkId)
				if err != nil && !errors.Is(err, model.ErrNotFound) {
					return nil, err
				}

				reader, err := info.a.getArtwork(ctx, info.artworkId, path, info.size)
				if err != nil {
					log.Error(ctx, "Error loading artwork art", "path", path, "size", info.size, err)
					return nil, err
				}
				return reader, nil
			})
	})
	return instanceImageCache
}
