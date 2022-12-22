package core

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
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

	// If requested a resized image, get the original (possibly from cache)
	if size > 0 && id != "" {
		r, err := a.Get(ctx, id, 0)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		resized, err := a.resizedFromOriginal(ctx, artID, r, size)
		return io.NopCloser(resized), err
	}

	key := &artworkKey{a: a, artID: artID, size: size}

	r, err := a.cache.Get(ctx, key)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Error(ctx, "Error accessing image cache", "id", id, "size", size, err)
	}
	return r, err
}

type fromFunc func() (io.ReadCloser, string)

func (f fromFunc) String() string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	name = strings.TrimPrefix(name, "github.com/navidrome/navidrome/core.")
	name = strings.TrimSuffix(name, ".func1")
	return name
}

func (a *artwork) get(ctx context.Context, artID model.ArtworkID, size int) (reader io.ReadCloser, path string, err error) {
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
	var ff = a.fromCoverArtPriority(ctx, conf.Server.CoverArtPriority, *al)
	ff = append(ff, fromPlaceholder())
	return extractImage(ctx, artID, ff...)
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
	if mf.CoverArtID().Kind == model.KindMediaFileArtwork {
		ff = []fromFunc{
			fromTag(mf.Path),
			fromFFmpegTag(ctx, a.ffmpeg, mf.Path),
		}
	}
	ff = append(ff, a.fromAlbum(ctx, mf.AlbumCoverArtID()))
	return extractImage(ctx, artID, ff...)
}

func (a *artwork) resizedFromOriginal(ctx context.Context, artID model.ArtworkID, original io.Reader, size int) (io.Reader, error) {
	// Keep a copy of the original data. In case we can't resize it, send it as is
	buf := new(bytes.Buffer)
	r := io.TeeReader(original, buf)

	resized, err := resizeImage(r, size)
	if err != nil {
		log.Warn(ctx, "Could not resize image. Will return image as is", "artID", artID, "size", size, err)
		// Force finish reading any remaining data
		_, _ = io.Copy(io.Discard, r)
		return buf, nil
	}
	return resized, nil
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

func (a *artwork) fromCoverArtPriority(ctx context.Context, priority string, al model.Album) []fromFunc {
	var ff []fromFunc
	for _, p := range strings.Split(strings.ToLower(priority), ",") {
		pat := strings.TrimSpace(p)
		if pat == "embedded" {
			ff = append(ff, fromTag(al.EmbedArtPath), fromFFmpegTag(ctx, a.ffmpeg, al.EmbedArtPath))
			continue
		}
		ff = append(ff, fromExternalFile(ctx, al.ImageFiles, p))
	}
	return ff
}

func fromExternalFile(ctx context.Context, files string, pattern string) fromFunc {
	return func() (io.ReadCloser, string) {
		for _, file := range filepath.SplitList(files) {
			_, name := filepath.Split(file)
			match, err := filepath.Match(pattern, strings.ToLower(name))
			if err != nil {
				log.Warn(ctx, "Error matching cover art file to pattern", "pattern", pattern, "file", file)
				continue
			}
			if !match {
				continue
			}
			f, err := os.Open(file)
			if err != nil {
				continue
			}
			return f, file
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

func asImageReader(r io.Reader) (io.Reader, string, error) {
	br := bufio.NewReader(r)
	buf, err := br.Peek(512)
	if err != nil {
		return nil, "", err
	}
	return br, http.DetectContentType(buf), nil
}

func resizeImage(reader io.Reader, size int) (io.Reader, error) {
	r, format, err := asImageReader(reader)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(r)
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
	buf.Reset()
	if format == "image/png" {
		err = png.Encode(buf, m)
	} else {
		err = jpeg.Encode(buf, m, &jpeg.Options{Quality: conf.Server.CoverJpegQuality})
	}
	return buf, err
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
	return fmt.Sprintf("%s.%d.%d", k.artID, k.size, conf.Server.CoverJpegQuality)
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
