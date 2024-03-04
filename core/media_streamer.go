package core

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/cache"
)

type MediaStreamer interface {
	NewStream(ctx context.Context, id string, reqFormat string, reqBitRate int, offset int) (*Stream, error)
	DoStream(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, reqOffset int) (*Stream, error)
}

type TranscodingCache cache.FileCache

func NewMediaStreamer(ds model.DataStore, t ffmpeg.FFmpeg, cache TranscodingCache) MediaStreamer {
	return &mediaStreamer{ds: ds, transcoder: t, cache: cache}
}

type mediaStreamer struct {
	ds         model.DataStore
	transcoder ffmpeg.FFmpeg
	cache      cache.FileCache
}

type streamJob struct {
	ms      *mediaStreamer
	mf      *model.MediaFile
	format  string
	bitRate int
	offset  int
}

func (j *streamJob) Key() string {
	return fmt.Sprintf("%s.%s.%d.%d.%s.%d", j.mf.ID, j.mf.UpdatedAt.Format(time.RFC3339Nano), j.mf.SubTrack, j.bitRate, j.format, j.offset)
}

func (j *streamJob) selectTargetFormat() string {
	if j.mf.Suffix == "wv" || j.mf.Suffix == "ape" || j.mf.Suffix == "flac" {
		// TODO: Decide by players supported lossless formats
		return "flac"
	}
	return conf.Server.DefaultDownsamplingFormat
}

func (ms *mediaStreamer) NewStream(ctx context.Context, id string, reqFormat string, reqBitRate int, reqOffset int) (*Stream, error) {
	mf, err := ms.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	return ms.DoStream(ctx, mf, reqFormat, reqBitRate, reqOffset)
}

func (ms *mediaStreamer) rawStream(stream *Stream, mf *model.MediaFile) (*Stream, error) {
	f, err := os.Open(mf.Path)
	if err != nil {
		return nil, err
	}
	stream.ReadCloser = f
	stream.Seeker = f
	stream.format = mf.Suffix
	return stream, nil
}

func (ms *mediaStreamer) allowStreaming(mf *model.MediaFile) bool {
	if mf.SubTrack > -1 {
		return false
	}
	// TODO: Decide on information from players supported format
	return mf.Suffix != "ape" && mf.Suffix != "wv"
}

func (ms *mediaStreamer) DoStream(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, reqOffset int) (*Stream, error) {
	var format string
	var bitRate int
	var cached bool
	defer func() {
		log.Info(ctx, "Streaming file", "title", mf.Title, "artist", mf.Artist, "format", format, "cached", cached,
			"bitRate", bitRate, "user", userName(ctx), "transcoding", format != mf.Suffix,
			"originalFormat", mf.Suffix, "originalBitRate", mf.BitRate)
	}()

	format, bitRate = selectTranscodingOptions(ctx, ms.ds, mf, reqFormat, reqBitRate)
	s := &Stream{ctx: ctx, mf: mf, format: format, bitRate: bitRate}

	if format == "raw" && ms.allowStreaming(mf) {
		log.Debug(ctx, "Streaming RAW file", "id", mf.ID, "path", mf.Path,
			"requestBitrate", reqBitRate, "requestFormat", reqFormat, "requestOffset", reqOffset,
			"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix,
			"selectedBitrate", bitRate, "selectedFormat", format)
		return ms.rawStream(s, mf)
	}

	job := &streamJob{
		ms:      ms,
		mf:      mf,
		format:  format,
		bitRate: bitRate,
		offset:  reqOffset,
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
		"requestBitrate", reqBitRate, "requestFormat", reqFormat, "requestOffset", reqOffset,
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
		return format, 0
	}
	if reqFormat == mf.Suffix && reqBitRate == 0 {
		bitRate = mf.BitRate
		return format, bitRate
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
		} else if reqBitRate > 0 && reqBitRate < mf.BitRate && conf.Server.DefaultDownsamplingFormat != "" {
			// If no format is specified and no transcoding associated to the player, but a bitrate is specified,
			// and there is no transcoding set for the player, we use the default downsampling format.
			// But only if the requested bitRate is lower than the original bitRate.
			log.Debug("Default Downsampling", "Using default downsampling format", conf.Server.DefaultDownsamplingFormat)
			cFormat = conf.Server.DefaultDownsamplingFormat
		}
	}
	if reqBitRate > 0 {
		cBitRate = reqBitRate
	}
	if cBitRate == 0 && cFormat == "" {
		return format, bitRate
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
	return format, bitRate
}

var (
	onceTranscodingCache     sync.Once
	instanceTranscodingCache TranscodingCache
)

func GetTranscodingCache() TranscodingCache {
	onceTranscodingCache.Do(func() {
		instanceTranscodingCache = NewTranscodingCache()
	})
	return instanceTranscodingCache
}

func NewTranscodingCache() TranscodingCache {
	return cache.NewFileCache("Transcoding", conf.Server.TranscodingCacheSize,
		consts.TranscodingCacheDir, consts.DefaultTranscodingCacheMaxItems,
		func(ctx context.Context, arg cache.Item) (io.Reader, error) {
			job := arg.(*streamJob)
			var t *model.Transcoding
			var err error

			if job.mf.SubTrack > -1 || job.format == "raw" {
				// Fake transcoder for transcode track media
				t = &model.Transcoding{
					Command:      ffmpeg.DefaultRawTranscodeCmd,
					TargetFormat: job.selectTargetFormat(),
				}
				log.Trace("Raw transcoding", "sourceFormat", job.mf.Suffix, "targetFormat", t.TargetFormat)
			} else {
				t, err = job.ms.ds.Transcoding(ctx).FindByFormat(job.format)
				if err != nil {
					log.Error(ctx, "Error loading transcoding command", "format", job.format, err)
					return nil, os.ErrInvalid
				}
			}
			out, err := job.ms.transcoder.Transcode(ctx, t.Command, t.TargetFormat, job.mf, job.bitRate, job.offset)
			if err != nil {
				log.Error(ctx, "Error starting transcoder", "id", job.mf.ID, err)
				return nil, os.ErrInvalid
			}
			return out, nil
		})
}
