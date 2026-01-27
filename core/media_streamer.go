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
	ms       *mediaStreamer
	mf       *model.MediaFile
	filePath string
	format   string
	bitRate  int
	offset   int
}

func (j *streamJob) Key() string {
	return fmt.Sprintf("%s.%s.%d.%s.%d", j.mf.ID, j.mf.UpdatedAt.Format(time.RFC3339Nano), j.bitRate, j.format, j.offset)
}

func (ms *mediaStreamer) NewStream(ctx context.Context, id string, reqFormat string, reqBitRate int, reqOffset int) (*Stream, error) {
	mf, err := ms.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	return ms.DoStream(ctx, mf, reqFormat, reqBitRate, reqOffset)
}

func (ms *mediaStreamer) DoStream(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, reqOffset int) (*Stream, error) {
	var format string
	var bitRate int
	var cached bool
	defer func() {
		log.Info(ctx, "Streaming file", "title", mf.Title, "artist", mf.Artist, "format", format, "cached", cached,
			"bitRate", bitRate, "user", userName(ctx), "transcoding", format != "raw",
			"originalFormat", mf.Suffix, "originalBitRate", mf.BitRate)
	}()

	format, bitRate = selectTranscodingOptions(ctx, ms.ds, mf, reqFormat, reqBitRate)
	s := &Stream{ctx: ctx, mf: mf, format: format, bitRate: bitRate}
	filePath := mf.AbsolutePath()

	if format == "raw" {
		log.Debug(ctx, "Streaming RAW file", "id", mf.ID, "path", filePath,
			"requestBitrate", reqBitRate, "requestFormat", reqFormat, "requestOffset", reqOffset,
			"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix,
			"selectedBitrate", bitRate, "selectedFormat", format)
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		s.ReadCloser = f
		s.Seeker = f
		s.format = mf.Suffix
		return s, nil
	}

	job := &streamJob{
		ms:       ms,
		mf:       mf,
		filePath: filePath,
		format:   format,
		bitRate:  bitRate,
		offset:   reqOffset,
	}
	r, err := ms.cache.Get(ctx, job)
	if err != nil {
		log.Error(ctx, "Error accessing transcoding cache", "id", mf.ID, err)
		return nil, err
	}
	cached = r.Cached

	s.ReadCloser = r
	s.Seeker = r.Seeker

	log.Debug(ctx, "Streaming TRANSCODED file", "id", mf.ID, "path", filePath,
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

func selectTranscodingOptions(ctx context.Context, ds model.DataStore, mf *model.MediaFile, reqFormat string, reqBitRate int) (format string, bitRate int) {
	// Default case
	format = "raw"
	bitRate = 0

	// If the client explicitly requests "raw"
	// then always serve the original
	if reqFormat == "raw" {
		return format, bitRate
	}

	// If requested format matches the file’s suffix and
	// no bitrate reduction is requested then
	// stream the file without transcoding
	if reqFormat == mf.Suffix && reqBitRate == 0 {
		return format, mf.BitRate
	}

	targetFormat, targetBitRate := findTargetTranscodingOptions(ctx, mf, reqFormat, reqBitRate)

	// If nothing was found then stream raw
	if targetFormat == "" && targetBitRate == 0 {
		return format, 0
	}

	t, err := ds.Transcoding(ctx).FindByFormat(targetFormat)
	if err != nil {
		// TODO: log error?
		return format, 0
	}

	format = t.TargetFormat

	// If no target bitrate was specified
	// fall back to the transcoding’s configuration
	// default bitrate
	if targetBitRate == 0 {
		bitRate = t.DefaultBitRate
	} else {
		bitRate = targetBitRate
	}

	// If the final format is the same as the original
	// and does not reduce bitrate
	// there’s no reason to transcode
	if format == mf.Suffix && bitRate >= mf.BitRate {
		return "raw", 0
	}

	return format, bitRate
}

func findTargetTranscodingOptions(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int) (string, int) {
	// If a format is requested use that
	if reqFormat != "" {
		return reqFormat, reqBitRate
	}

	// If a default transcoding configuration exists for this context
	if trc, ok := request.TranscodingFrom(ctx); ok {
		targetFormat := trc.TargetFormat
		targetBitRate := trc.DefaultBitRate

		// If a player is configured adjust bitrate based on
		// user request or player limits
		if p, hasPlayer := request.PlayerFrom(ctx); hasPlayer {
			if reqBitRate > 0 {
				targetBitRate = reqBitRate
			} else if p.MaxBitRate > 0 {
				targetBitRate = p.MaxBitRate
			}
		} else if reqBitRate > 0 {
			targetBitRate = reqBitRate
		}

		return targetFormat, targetBitRate
	}

	// Use the default downsampling format the server is configured to but
	// only if the requested bitrate is reduced
	isBitrateReduced := reqBitRate > 0 && reqBitRate < mf.BitRate
	hasDefaultDownsamplingFormat := conf.Server.DefaultDownsamplingFormat != ""

	if isBitrateReduced && hasDefaultDownsamplingFormat {
		log.Debug("Default Downsampling",
			"Using default downsampling format",
			conf.Server.DefaultDownsamplingFormat)
		return conf.Server.DefaultDownsamplingFormat, reqBitRate
	}

	return "", 0
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
			t, err := job.ms.ds.Transcoding(ctx).FindByFormat(job.format)
			if err != nil {
				log.Error(ctx, "Error loading transcoding command", "format", job.format, err)
				return nil, os.ErrInvalid
			}

			// Choose the appropriate context based on EnableTranscodingCancellation configuration.
			// This is where we decide whether transcoding processes should be cancellable or not.
			var transcodingCtx context.Context
			if conf.Server.EnableTranscodingCancellation {
				// Use the request context directly, allowing cancellation when client disconnects
				transcodingCtx = ctx
			} else {
				// Use background context with request values preserved.
				// This prevents cancellation but maintains request metadata (user, client, etc.)
				transcodingCtx = request.AddValues(context.Background(), ctx)
			}

			out, err := job.ms.transcoder.Transcode(transcodingCtx, t.Command, job.filePath, job.bitRate, job.offset)
			if err != nil {
				log.Error(ctx, "Error starting transcoder", "id", job.mf.ID, err)
				return nil, os.ErrInvalid
			}
			return out, nil
		})
}
