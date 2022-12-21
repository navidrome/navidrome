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
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/dhowden/tag"
	"github.com/disintegration/imaging"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/singleton"
	_ "golang.org/x/image/webp"
)

type Artwork interface {
	Get(ctx context.Context, id string, size int) (io.ReadCloser, error)
}

func NewArtwork(ds model.DataStore, cache cache.FileCache, ffmpeg ffmpeg.FFmpeg) Artwork {
	return &artwork{ds: ds, cache: cache, ffmpeg: ffmpeg}
}

type artwork struct {
	ds     model.DataStore
	cache  cache.FileCache
	ffmpeg ffmpeg.FFmpeg
}

func (a *artwork) Get(ctx context.Context, id string, size int) (io.ReadCloser, error) {
	var artID model.ArtworkID
	var err error
	if id != "" {
		artID, err = model.ParseArtworkID(id)
		if err != nil {
			return nil, errors.New("invalid ID")
		}
	}

	key := &artworkKey{a: a, artID: artID, size: size}

	r, err := a.cache.Get(ctx, key)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Error(ctx, "Error accessing image cache", "id", id, "size", size, err)
	}
	return r, err
}

func (a *artwork) get(ctx context.Context, artID model.ArtworkID, size int) (reader io.ReadCloser, path string, err error) {
	// If requested a resized image
	if size > 0 {
		return a.resizedFromOriginal(ctx, artID, size)
	}

	switch artID.Kind {
	case model.KindAlbumArtwork:
		reader, path = a.extractAlbumImage(ctx, artID)
	case model.KindMediaFileArtwork:
		reader, path = a.extractMediaFileImage(ctx, artID)
	default:
		reader, path = fromPlaceholder()()
	}
	return reader, path, ctx.Err()
}

func (a *artwork) extractAlbumImage(ctx context.Context, artID model.ArtworkID) (io.ReadCloser, string) {
	al, err := a.ds.Album(ctx).Get(artID.ID)
	if errors.Is(err, model.ErrNotFound) {
		r, path := fromPlaceholder()()
		return r, path
	}
	if err != nil {
		log.Error(ctx, "Could not retrieve album", "id", artID.ID, err)
		return nil, ""
	}

	return extractImage(ctx, artID,
		fromExternalFile(al.ImageFiles, "cover.png", "cover.jpg", "cover.jpeg", "cover.webp"),
		fromExternalFile(al.ImageFiles, "folder.png", "folder.jpg", "folder.jpeg", "folder.webp"),
		fromExternalFile(al.ImageFiles, "album.png", "album.jpg", "album.jpeg", "album.webp"),
		fromExternalFile(al.ImageFiles, "albumart.png", "albumart.jpg", "albumart.jpeg", "albumart.webp"),
		fromExternalFile(al.ImageFiles, "front.png", "front.jpg", "front.jpeg", "front.webp"),
		fromTag(al.EmbedArtPath),
		fromFFmpegTag(ctx, a.ffmpeg, al.EmbedArtPath),
		fromPlaceholder(),
	)
}

func (a *artwork) extractMediaFileImage(ctx context.Context, artID model.ArtworkID) (reader io.ReadCloser, path string) {
	mf, err := a.ds.MediaFile(ctx).Get(artID.ID)
	if errors.Is(err, model.ErrNotFound) {
		r, path := fromPlaceholder()()
		return r, path
	}
	if err != nil {
		log.Error(ctx, "Could not retrieve mediafile", "id", artID.ID, err)
		return nil, ""
	}

	var ff []fromFunc
	if mf.HasCoverArt {
		ff = []fromFunc{
			fromTag(mf.Path),
			fromFFmpegTag(ctx, a.ffmpeg, mf.Path),
		}
	}
	ff = append(ff, a.fromAlbum(ctx, mf.AlbumCoverArtID()))
	return extractImage(ctx, artID, ff...)
}

func (a *artwork) resizedFromOriginal(ctx context.Context, artID model.ArtworkID, size int) (io.ReadCloser, string, error) {
	r, path, err := a.get(ctx, artID, 0)
	if err != nil || r == nil {
		return nil, "", err
	}
	defer r.Close()
	usePng := strings.ToLower(filepath.Ext(path)) == ".png"
	r, err = resizeImage(r, size, usePng)
	if err != nil {
		log.Warn(ctx, "Could not resize image. Using placeholder", "artID", artID, "size", size, err)
		r, path := fromPlaceholder()()
		return r, path, nil
	}
	return r, fmt.Sprintf("%s@%d", path, size), nil
}

type fromFunc func() (io.ReadCloser, string)

func (f fromFunc) String() string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	name = strings.TrimPrefix(name, "github.com/navidrome/navidrome/core.")
	name = strings.TrimSuffix(name, ".func1")
	return name
}

func extractImage(ctx context.Context, artID model.ArtworkID, extractFuncs ...fromFunc) (io.ReadCloser, string) {
	for _, f := range extractFuncs {
		if ctx.Err() != nil {
			return nil, ""
		}
		r, path := f()
		if r != nil {
			log.Trace(ctx, "Found artwork", "artID", artID, "path", path, "origin", f)
			return r, path
		}
	}
	log.Error(ctx, "extractImage should never reach this point!", "artID", artID, "path")
	return nil, ""
}

func (a *artwork) fromAlbum(ctx context.Context, id model.ArtworkID) fromFunc {
	return func() (io.ReadCloser, string) {
		r, path, err := a.get(ctx, id, 0)
		if err != nil {
			return nil, ""
		}
		return r, path
	}
}

// This is a bit unoptimized, but we need to make sure the priority order of validNames
// is preserved (i.e. png is better than jpg)
func fromExternalFile(files string, validNames ...string) fromFunc {
	return func() (io.ReadCloser, string) {
		fileList := filepath.SplitList(files)
		for _, validName := range validNames {
			for _, file := range fileList {
				_, name := filepath.Split(file)
				if !strings.EqualFold(validName, name) {
					continue
				}
				f, err := os.Open(file)
				if err != nil {
					continue
				}
				return f, file
			}
		}
		return nil, ""
	}
}

func fromTag(path string) fromFunc {
	return func() (io.ReadCloser, string) {
		if path == "" {
			return nil, ""
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, ""
		}
		defer f.Close()

		m, err := tag.ReadFrom(f)
		if err != nil {
			return nil, ""
		}

		picture := m.Picture()
		if picture == nil {
			return nil, ""
		}
		return io.NopCloser(bytes.NewReader(picture.Data)), path
	}
}

func fromFFmpegTag(ctx context.Context, ffmpeg ffmpeg.FFmpeg, path string) fromFunc {
	return func() (io.ReadCloser, string) {
		if path == "" {
			return nil, ""
		}
		r, err := ffmpeg.ExtractImage(ctx, path)
		if err != nil {
			return nil, ""
		}
		defer r.Close()
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, r)
		if err != nil {
			return nil, ""
		}
		return io.NopCloser(buf), path
	}
}

func fromPlaceholder() fromFunc {
	return func() (io.ReadCloser, string) {
		r, _ := resources.FS().Open(consts.PlaceholderAlbumArt)
		return r, consts.PlaceholderAlbumArt
	}
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

type ArtworkCache struct {
	cache.FileCache
}

type artworkKey struct {
	a     *artwork
	artID model.ArtworkID
	size  int
}

func (k *artworkKey) Key() string {
	return fmt.Sprintf("%s.%d.%d.%d", k.artID.ID, k.size, k.artID.LastUpdate.UnixNano(), conf.Server.CoverJpegQuality)
}

func GetImageCache() cache.FileCache {
	return singleton.GetInstance(func() *ArtworkCache {
		return &ArtworkCache{
			FileCache: cache.NewFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems,
				func(ctx context.Context, arg cache.Item) (io.Reader, error) {
					info := arg.(*artworkKey)
					r, _, err := info.a.get(ctx, info.artID, info.size)
					return r, err
				}),
		}
	})
}
