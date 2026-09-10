package artwork

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/httpclient"
	"go.senan.xyz/taglib"
)

// errSourceUnreadable marks a candidate the resolver knows exists but could not read. Failing
// to open it is not evidence the entity has no artwork, so callers must not settle on absent.
var errSourceUnreadable = errors.New("artwork source unreadable")

type sourceFunc func() (r io.ReadCloser, path string, err error)

func fromExternalFile(ctx context.Context, libFS fs.FS, files []string, pattern string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		var openErr error
		for _, file := range files {
			_, name := filepath.Split(file)
			match, err := filepath.Match(pattern, strings.ToLower(name))
			if err != nil {
				log.Warn(ctx, "Artwork: Error matching cover art file to pattern", "pattern", pattern, "file", file)
				continue
			}
			if !match {
				continue
			}
			f, err := libFS.Open(file)
			if err != nil {
				log.Warn(ctx, "Artwork: Could not open cover art file", "file", file, err)
				openErr = fmt.Errorf("%w: %s: %w", errSourceUnreadable, file, err)
				continue
			}
			return f, file, nil
		}
		if openErr != nil {
			return nil, "", openErr
		}
		return nil, "", fmt.Errorf("pattern '%s' not matched by files %v", pattern, files)
	}
}

// These regexes are used to match the picture type in the file, in the order they are listed.
var picTypeRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?i).*cover.*front.*|.*front.*cover.*`),
	regexp.MustCompile(`(?i).*front.*`),
	regexp.MustCompile(`(?i).*cover.*`),
}

func fromTag(ctx context.Context, libFS fs.FS, relPath string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		if relPath == "" {
			return nil, "", nil
		}
		f, err := libFS.Open(relPath)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %s: %w", errSourceUnreadable, relPath, err)
		}
		rs, ok := f.(io.ReadSeeker)
		if !ok {
			f.Close()
			return nil, "", fmt.Errorf("FS file %s is not seekable; cannot read tags", relPath)
		}
		tf, err := taglib.OpenStream(rs,
			taglib.WithReadStyle(taglib.ReadStyleFast),
			taglib.WithFilename(relPath),
		)
		if err != nil {
			f.Close()
			return nil, "", fmt.Errorf("%w: %s: %w", errSourceUnreadable, relPath, err)
		}
		// Close in LIFO order: tf first (it holds rs internally), then f.
		defer f.Close()
		defer tf.Close()

		images := tf.Properties().Images
		if len(images) == 0 {
			return nil, "", fmt.Errorf("no embedded image found in %s", relPath)
		}

		imageIndex := findBestImageIndex(ctx, images, relPath)
		data, err := tf.Image(imageIndex)
		if err != nil || len(data) == 0 {
			return nil, "", fmt.Errorf("could not load embedded image from %s", relPath)
		}
		return io.NopCloser(bytes.NewReader(data)), relPath, nil
	}
}

func findBestImageIndex(ctx context.Context, images []taglib.ImageDesc, path string) int {
	for _, regex := range picTypeRegexes {
		for i, img := range images {
			if regex.MatchString(img.Type) {
				log.Trace(ctx, "Artwork: Found embedded image", "type", img.Type, "path", path)
				return i
			}
		}
	}
	log.Trace(ctx, "Artwork: Could not find a front image. Getting the first one", "type", images[0].Type, "path", path)
	return 0
}

// fromFFmpegTag is intentionally absolute-path-based. ffmpeg is a subprocess
// and cannot read from arbitrary fs.FS implementations; piping via stdin is a
// non-trivial refactor with stream/seek implications.
//
// TODO(artwork-musicfs): when the storage backing the library is not local
// (e.g. a future S3 backend, or FakeFS in tests), short-circuit this source
// func to return (nil, "", nil) so callers fall through cleanly.
func fromFFmpegTag(ctx context.Context, ffmpeg ffmpeg.FFmpeg, path string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		if path == "" {
			return nil, "", nil
		}
		r, err := ffmpeg.ExtractImage(ctx, path)
		if err != nil {
			return nil, "", err
		}
		// Validate that the stream actually contains image data by reading the first byte.
		// ffmpeg.ExtractImage returns a pipe reader that may fail asynchronously if the
		// file has no video/image stream (e.g., an MP3 without embedded art).
		buf := make([]byte, 1)
		n, err := r.Read(buf)
		if n == 0 || err != nil {
			r.Close()
			return nil, "", fmt.Errorf("ffmpeg produced no image data for %s: %w", path, err)
		}
		return readCloser{Reader: io.MultiReader(bytes.NewReader(buf[:n]), r), Closer: r}, path, nil
	}
}

// readCloser combines a Reader and a Closer into an io.ReadCloser.
type readCloser struct {
	io.Reader
	io.Closer
}

func fromURL(ctx context.Context, imageUrl *url.URL) (io.ReadCloser, string, error) {
	hc := httpclient.New(5 * time.Second)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, imageUrl.String(), nil)
	resp, err := hc.Do(req) //nolint:gosec
	if err != nil {
		return nil, "", err
	}
	// An agent-advertised URL that 404s is a definitive miss, not a fault: settle absent
	// instead of retrying forever and tripping the artwork breaker.
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		resp.Body.Close()
		return nil, "", model.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("error retrieving artwork from %s: %s", imageUrl, resp.Status)
	}
	return resp.Body, imageUrl.String(), nil
}
