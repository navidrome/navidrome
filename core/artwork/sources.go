package artwork

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
)

func selectImageReader(ctx context.Context, artID model.ArtworkID, extractFuncs ...sourceFunc) (io.ReadCloser, string, error) {
	for _, f := range extractFuncs {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		start := time.Now()
		r, path, err := f()
		if r != nil {
			msg := fmt.Sprintf("Found %s artwork", artID.Kind)
			log.Debug(ctx, msg, "artID", artID, "path", path, "source", f, "elapsed", time.Since(start))
			return r, path, nil
		}
		log.Trace(ctx, "Failed trying to extract artwork", "artID", artID, "source", f, "elapsed", time.Since(start), err)
	}
	return nil, "", fmt.Errorf("could not get `%s` cover art for %s: %w", artID.Kind, artID, ErrUnavailable)
}

type sourceFunc func() (r io.ReadCloser, path string, err error)

func (f sourceFunc) String() string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	name = strings.TrimPrefix(name, "github.com/navidrome/navidrome/core/artwork.")
	if _, after, found := strings.Cut(name, ")."); found {
		name = after
	}
	name = strings.TrimSuffix(name, ".func1")
	return name
}

func splitList(s string) []string {
	return strings.Split(s, consts.Zwsp)
}

func fromExternalFile(ctx context.Context, files string, pattern string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		for _, file := range splitList(files) {
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
				log.Warn(ctx, "Could not open cover art file", "file", file, err)
				continue
			}
			return f, file, err
		}
		return nil, "", fmt.Errorf("pattern '%s' not matched by files %v", pattern, files)
	}
}

func fromTag(path string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		if path == "" {
			return nil, "", nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}
		defer f.Close()

		m, err := tag.ReadFrom(f)
		if err != nil {
			return nil, "", err
		}

		picture := m.Picture()
		if picture == nil {
			return nil, "", fmt.Errorf("no embedded image found in %s", path)
		}
		return io.NopCloser(bytes.NewReader(picture.Data)), path, nil
	}
}

func fromFFmpegTag(ctx context.Context, ffmpeg ffmpeg.FFmpeg, path string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		if path == "" {
			return nil, "", nil
		}
		r, err := ffmpeg.ExtractImage(ctx, path)
		if err != nil {
			return nil, "", err
		}
		defer r.Close()
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, r)
		if err != nil {
			return nil, "", err
		}
		return io.NopCloser(buf), path, nil
	}
}

func fromAlbum(ctx context.Context, a *artwork, id model.ArtworkID) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		r, _, err := a.Get(ctx, id, 0, false)
		if err != nil {
			return nil, "", err
		}
		return r, id.String(), nil
	}
}

func fromAlbumPlaceholder() sourceFunc {
	return func() (io.ReadCloser, string, error) {
		r, _ := resources.FS().Open(consts.PlaceholderAlbumArt)
		return r, consts.PlaceholderAlbumArt, nil
	}
}
func fromArtistExternalSource(ctx context.Context, ar model.Artist, em core.ExternalMetadata) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		imageUrl, err := em.ArtistImage(ctx, ar.ID)
		if err != nil {
			return nil, "", err
		}

		return fromURL(ctx, imageUrl)
	}
}

func fromAlbumExternalSource(ctx context.Context, al model.Album, em core.ExternalMetadata) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		imageUrl, err := em.AlbumImage(ctx, al.ID)
		if err != nil {
			return nil, "", err
		}

		return fromURL(ctx, imageUrl)
	}
}

func fromURL(ctx context.Context, imageUrl *url.URL) (io.ReadCloser, string, error) {
	hc := http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, imageUrl.String(), nil)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("error retrieveing artwork from %s: %s", imageUrl, resp.Status)
	}
	return resp.Body, imageUrl.String(), nil
}
