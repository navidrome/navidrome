package core

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/transcoder"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/cache"
)

type MediaStreamer interface {
	NewStream(ctx context.Context, id string, reqFormat string, reqBitRate int) (*Stream, error)
}

type TranscodingCache cache.FileCache

func NewMediaStreamer(ds model.DataStore, fsys fs.FS, ffm transcoder.Transcoder, cache TranscodingCache) MediaStreamer {
	return &mediaStreamer{ds: ds, fsys: fsys, ffm: ffm, cache: cache}
}

type mediaStreamer struct {
	ds    model.DataStore
	fsys  fs.FS
	ffm   transcoder.Transcoder
	cache cache.FileCache
}

type streamJob struct {
	ms      *mediaStreamer
	mf      *model.MediaFile
	format  string
	bitRate int
}

func (j *streamJob) Key() string {
	return fmt.Sprintf("%s.%s.%d.%s", j.mf.ID, j.mf.UpdatedAt.Format(time.RFC3339Nano), j.bitRate, j.format)
}

func (ms *mediaStreamer) NewStream(ctx context.Context, id string, reqFormat string, reqBitRate int) (*Stream, error) {
	mf, err := ms.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	var format string
	var bitRate int
	var cached bool
	defer func() {
		log.Info("Streaming file", "title", mf.Title, "artist", mf.Artist, "format", format, "cached", cached,
			"bitRate", bitRate, "user", userName(ctx), "transcoding", format != "raw",
			"originalFormat", mf.Suffix, "originalBitRate", mf.BitRate)
	}()

	format, bitRate = selectTranscodingOptions(ctx, ms.ds, mf, reqFormat, reqBitRate)
	s := &Stream{ctx: ctx, mf: mf, format: format, bitRate: bitRate}

	if format == "raw" {
		log.Debug(ctx, "Streaming RAW file", "id", mf.ID, "path", mf.Path,
			"requestBitrate", reqBitRate, "requestFormat", reqFormat,
			"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix,
			"selectedBitrate", bitRate, "selectedFormat", format)

		f, err := ms.fsys.Open(mf.Path)
		if err != nil {
			return nil, err
		}

		rsc, ok := f.(io.ReadSeeker)
		if !ok {
			return nil, fmt.Errorf("file interface does not implement io.ReadSeeker")
		}

		s.ReadCloser = f
		s.Seeker = rsc
		s.format = mf.Suffix
		return s, nil
	}

	job := &streamJob{
		ms:      ms,
		mf:      mf,
		format:  format,
		bitRate: bitRate,
	}
	r, err := ms.cache.Get(ctx, job)
	if err != nil {
		log.Error(ctx, "Error accessing transcoding cache", "id", mf.ID, err)
		return nil, err
	}
	cached = r.Cached

	s.ReadCloser = r
	s.Seeker = r.Seeker

	log.Debug(ctx, "Streaming TRANSCODED file", "id", mf.ID, "path", mf.Path,
		"requestBitrate", reqBitRate, "requestFormat", reqFormat,
		"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix,
		"selectedBitrate", bitRate, "selectedFormat", format, "cached", cached, "seekable", s.Seekable())

	return s, nil
}

type Stream struct {
	ctx     context.Context
	mf      *model.MediaFile
	bitRate int
	format  string
	io.ReadCloser
	io.Seeker
}

func (s *Stream) Seekable() bool      { return s.Seeker != nil }
func (s *Stream) Duration() float32   { return s.mf.Duration }
func (s *Stream) ContentType() string { return mime.TypeByExtension("." + s.format) }
func (s *Stream) Name() string        { return s.mf.Title + "." + s.format }
func (s *Stream) ModTime() time.Time  { return s.mf.UpdatedAt }
func (s *Stream) EstimatedContentLength() int {
	return int(s.mf.Duration * float32(s.bitRate) / 8 * 1024)
}

// TODO This function deserves some love (refactoring)
func selectTranscodingOptions(ctx context.Context, ds model.DataStore, mf *model.MediaFile, reqFormat string, reqBitRate int) (format string, bitRate int) {
	format = "raw"
	if reqFormat == "raw" {
		return
	}
	if reqFormat == mf.Suffix && reqBitRate == 0 {
		bitRate = mf.BitRate
		return
	}
	trc, hasDefault := request.TranscodingFrom(ctx)
	var cFormat string
	var cBitRate int
	if reqFormat != "" {
		cFormat = reqFormat
	} else {
		if hasDefault {
			cFormat = trc.TargetFormat
			cBitRate = trc.DefaultBitRate
			if p, ok := request.PlayerFrom(ctx); ok {
				cBitRate = p.MaxBitRate
			}
		}
	}
	if reqBitRate > 0 {
		cBitRate = reqBitRate
	}
	if cBitRate == 0 && cFormat == "" {
		return
	}
	t, err := ds.Transcoding(ctx).FindByFormat(cFormat)
	if err == nil {
		format = t.TargetFormat
		if cBitRate != 0 {
			bitRate = cBitRate
		} else {
			bitRate = t.DefaultBitRate
		}
	}
	if format == mf.Suffix && bitRate >= mf.BitRate {
		format = "raw"
		bitRate = 0
	}
	return
}

var (
	onceTranscodingCache     sync.Once
	instanceTranscodingCache TranscodingCache
)

func GetTranscodingCache() TranscodingCache {
	onceTranscodingCache.Do(func() {
		instanceTranscodingCache = cache.NewFileCache("Transcoding", conf.Server.TranscodingCacheSize,
			consts.TranscodingCacheDir, consts.DefaultTranscodingCacheMaxItems,
			func(ctx context.Context, arg cache.Item) (io.Reader, error) {
				job := arg.(*streamJob)
				t, err := job.ms.ds.Transcoding(ctx).FindByFormat(job.format)
				if err != nil {
					log.Error(ctx, "Error loading transcoding command", "format", job.format, err)
					return nil, os.ErrInvalid
				}
				out, err := job.ms.ffm.Start(ctx, t.Command, job.mf.Path, job.bitRate)
				if err != nil {
					log.Error(ctx, "Error starting transcoder", "id", job.mf.ID, err)
					return nil, os.ErrInvalid
				}
				return out, nil
			})
	})
	return instanceTranscodingCache
}
