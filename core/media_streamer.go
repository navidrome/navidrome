package core

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/core/transcoder"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
)

type MediaStreamer interface {
	NewStream(ctx context.Context, id string, reqFormat string, reqBitRate int) (*Stream, error)
}

type TranscodingCache FileCache

func NewMediaStreamer(ds model.DataStore, ffm transcoder.Transcoder, cache TranscodingCache) MediaStreamer {
	return &mediaStreamer{ds: ds, ffm: ffm, cache: cache}
}

type mediaStreamer struct {
	ds    model.DataStore
	ffm   transcoder.Transcoder
	cache FileCache
}

type streamJob struct {
	ms      *mediaStreamer
	mf      *model.MediaFile
	format  string
	bitRate int
}

func (j *streamJob) String() string {
	return fmt.Sprintf("%s.%d.%s", j.mf.ID, j.bitRate, j.format)
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
		f, err := os.Open(mf.Path)
		if err != nil {
			return nil, err
		}
		s.Reader = f
		s.Closer = f
		s.Seeker = f
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

	log.Debug(ctx, "Streaming TRANSCODED file", "id", mf.ID, "path", mf.Path,
		"requestBitrate", reqBitRate, "requestFormat", reqFormat,
		"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix,
		"selectedBitrate", bitRate, "selectedFormat", format, "cached", cached)

	s.Reader = r
	s.Closer = r
	if r.Seekable() {
		s.Seeker = r
	}

	return s, nil
}

type Stream struct {
	ctx     context.Context
	mf      *model.MediaFile
	bitRate int
	format  string
	io.Reader
	io.Closer
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

func NewTranscodingCache() TranscodingCache {
	return NewFileCache("Transcoding", conf.Server.TranscodingCacheSize,
		consts.TranscodingCacheDir, consts.DefaultTranscodingCacheMaxItems,
		func(ctx context.Context, arg fmt.Stringer) (io.Reader, error) {
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
}
