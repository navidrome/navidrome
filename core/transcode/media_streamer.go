package transcode

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"strings"
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
	NewStream(ctx context.Context, req StreamRequest) (*Stream, error)
	DoStream(ctx context.Context, mf *model.MediaFile, req StreamRequest) (*Stream, error)
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
	ms         *mediaStreamer
	mf         *model.MediaFile
	filePath   string
	format     string
	bitRate    int
	sampleRate int
	bitDepth   int
	channels   int
	offset     int
}

func (j *streamJob) Key() string {
	return fmt.Sprintf("%s.%s.%d.%d.%d.%d.%s.%d", j.mf.ID, j.mf.UpdatedAt.Format(time.RFC3339Nano), j.bitRate, j.sampleRate, j.bitDepth, j.channels, j.format, j.offset)
}

func (ms *mediaStreamer) NewStream(ctx context.Context, req StreamRequest) (*Stream, error) {
	mf, err := ms.ds.MediaFile(ctx).Get(req.ID)
	if err != nil {
		return nil, err
	}

	return ms.DoStream(ctx, mf, req)
}

func (ms *mediaStreamer) DoStream(ctx context.Context, mf *model.MediaFile, req StreamRequest) (*Stream, error) {
	var format string
	var bitRate int
	var cached bool
	defer func() {
		log.Info(ctx, "Streaming file", "title", mf.Title, "artist", mf.Artist, "format", format, "cached", cached,
			"bitRate", bitRate, "sampleRate", req.SampleRate, "bitDepth", req.BitDepth, "channels", req.Channels,
			"user", userName(ctx), "transcoding", format != "raw",
			"originalFormat", mf.Suffix, "originalBitRate", mf.BitRate)
	}()

	format = req.Format
	bitRate = req.BitRate
	if format == "" || format == "raw" {
		format = "raw"
		bitRate = 0
	}
	s := &Stream{ctx: ctx, mf: mf, format: format, bitRate: bitRate}
	filePath := mf.AbsolutePath()

	if format == "raw" {
		log.Debug(ctx, "Streaming RAW file", "id", mf.ID, "path", filePath,
			"requestBitrate", req.BitRate, "requestFormat", req.Format, "requestOffset", req.Offset,
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
		ms:         ms,
		mf:         mf,
		filePath:   filePath,
		format:     format,
		bitRate:    bitRate,
		sampleRate: req.SampleRate,
		bitDepth:   req.BitDepth,
		channels:   req.Channels,
		offset:     req.Offset,
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
		"requestBitrate", req.BitRate, "requestFormat", req.Format, "requestOffset", req.Offset,
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

// NewTestStream creates a Stream for testing purposes.
func NewTestStream(mf *model.MediaFile, format string, bitRate int) *Stream {
	return &Stream{
		ctx:        context.Background(),
		mf:         mf,
		format:     format,
		bitRate:    bitRate,
		ReadCloser: io.NopCloser(strings.NewReader("")),
	}
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
			command := LookupTranscodeCommand(ctx, job.ms.ds, job.format)
			if command == "" {
				log.Error(ctx, "No transcoding command available", "format", job.format)
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

			out, err := job.ms.transcoder.Transcode(transcodingCtx, ffmpeg.TranscodeOptions{
				Command:    command,
				Format:     job.format,
				FilePath:   job.filePath,
				BitRate:    job.bitRate,
				SampleRate: job.sampleRate,
				BitDepth:   job.bitDepth,
				Channels:   job.channels,
				Offset:     job.offset,
			})
			if err != nil {
				log.Error(ctx, "Error starting transcoder", "id", job.mf.ID, err)
				return nil, os.ErrInvalid
			}
			return out, nil
		})
}

// userName extracts the username from the context for logging purposes.
func userName(ctx context.Context) string {
	if user, ok := request.UserFrom(ctx); !ok {
		return "UNKNOWN"
	} else {
		return user.UserName
	}
}
