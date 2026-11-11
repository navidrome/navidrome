package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/req"
)

type MediaStreamer interface {
	NewStream(ctx context.Context, mf *model.MediaFile, req Request) (*Stream, error)
}

type TranscodingCache cache.FileCache

func NewMediaStreamer(ds model.DataStore, t ffmpeg.FFmpeg, cache TranscodingCache) MediaStreamer {
	return &mediaStreamer{
		ds:         ds,
		transcoder: t,
		cache:      cache,
		limiter:    NewTranscodeLimiter(conf.Server.Transcoding.MaxConcurrent, conf.Server.Transcoding.MaxConcurrentPerUser),
	}
}

type mediaStreamer struct {
	ds         model.DataStore
	transcoder ffmpeg.FFmpeg
	cache      cache.FileCache
	limiter    TranscodeLimiter
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

// NewStream creates a Stream for the given MediaFile and Request. It handles both raw streaming (no transcoding)
// and transcoded streaming based on the requested format and bitrate. It also logs detailed information about
// the streaming request and whether the transcoding result was served from cache or not.
func (ms *mediaStreamer) NewStream(ctx context.Context, mf *model.MediaFile, req Request) (*Stream, error) {
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
	s := &Stream{ctx: ctx, mf: mf, format: format, bitRate: bitRate, offset: req.Offset}
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
		// Rate-limit rejections are already logged at warn level by the
		// producer; treating them as cache failures here would both
		// double-log and mask actual cache problems.
		if !errors.Is(err, ErrTooManyTranscodes) {
			log.Error(ctx, "Error accessing transcoding cache", "id", mf.ID, err)
		}
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
	ctx     context.Context //nolint:containedctx // stream outlives the call that built it; Read has no ctx
	mf      *model.MediaFile
	bitRate int
	format  string
	offset  int
	io.ReadCloser
	io.Seeker
}

func (s *Stream) Seekable() bool      { return s.Seeker != nil }
func (s *Stream) Duration() float32   { return s.mf.Duration }
func (s *Stream) ContentType() string { return mime.TypeByExtension("." + s.format) }
func (s *Stream) Name() string        { return s.mf.Title + "." + s.format }
func (s *Stream) ModTime() time.Time  { return s.mf.UpdatedAt }

// estimateHeadroom inflates the declared Content-Length when DevMatchEstimatedContentLength is on, so
// output over the estimate is padded instead of cut. Measured MP3 to Opus outputs (VBR) came in at up
// to 1.074 of EstimatedContentLength. Without padding, a larger estimate only makes the body come up
// short more often, so the headroom goes with the flag.
const estimateHeadroom = 1.10

// EstimatedContentLength returns the expected size of the transcoded output, from the part of the track
// after the requested offset. It returns 0 when the track has no duration left to estimate from.
func (s *Stream) EstimatedContentLength() int64 {
	remaining := max(float64(s.mf.Duration)-float64(s.offset), 0)
	return int64(remaining * float64(s.bitRate) / 8 * 1024)
}

// Serve writes the stream to the HTTP response. For seekable streams it uses http.ServeContent
// (supporting range requests). For non-seekable streams it writes directly and logs any errors.
// Returns the number of bytes written, or -1 when it does not count them (http.ServeContent and HEAD
// requests), and an error only when it fails with 0 bytes written
// (meaning the HTTP 200 status has not been flushed yet and the caller can still send an error response).
// Once bytes are on the wire it panics with http.ErrAbortHandler instead, aborting the response.
// Empty output (0 bytes, no error) is logged but not treated as an error.
// With estimateContentLength=true, a non-seekable stream declares an estimated Content-Length. With
// DevMatchEstimatedContentLength, the body is also cut or zero-padded to match it, so clients never
// see a body that disagrees with the header.
func (s *Stream) Serve(ctx context.Context, w http.ResponseWriter, r *http.Request) (int64, error) {
	if s.Seekable() {
		http.ServeContent(w, r, s.Name(), s.ModTime(), s)
		return -1, nil
	}

	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("Content-Type", s.ContentType())

	// limit > 0 only when the body must match the declared Content-Length.
	var limit int64
	if req.Params(r).BoolOr("estimateContentLength", false) {
		length := s.EstimatedContentLength()
		if conf.Server.DevMatchEstimatedContentLength {
			length = int64(float64(length) * estimateHeadroom)
			limit = length
		}
		if length > 0 {
			log.Trace(ctx, "Estimated content-length", "contentLength", length)
			w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		}
	}

	if r.Method == http.MethodHead {
		go func() { _, _ = io.Copy(io.Discard, s) }()
		return -1, nil
	}

	var src io.Reader = s
	if limit > 0 {
		src = io.LimitReader(s, limit)
	}

	id := s.mf.ID
	c, err := io.Copy(w, src)
	if err != nil {
		log.Error(ctx, "Error sending transcoded file", "id", id, err)
		if c == 0 {
			w.Header().Del("Content-Length")
			return 0, fmt.Errorf("sending transcoded file: %w", err)
		}
		// The 200 is already sent, so dropping the connection is the only way to say "truncated".
		panic(http.ErrAbortHandler)
	}
	if c == 0 {
		// Nothing is sent yet, so drop the estimate instead of padding a body that holds no audio.
		w.Header().Del("Content-Length")
		log.Error(ctx, "Transcoding returned empty output, ffmpeg may have failed. "+
			"Check that ffmpeg supports the requested codec. Enable Trace logging for ffmpeg stderr details",
			"id", id, "format", s.ContentType())
		return 0, nil
	}
	switch {
	case c < limit:
		padding, err := io.CopyN(w, zeroReader{}, limit-c)
		if err != nil {
			log.Debug(ctx, "Error padding transcoded file to the estimated length", "id", id, err)
		}
		c += padding
	case limit > 0:
		// The body is complete. Send it now, then read the rest, so a transcode tied to this request
		// can finish and complete its cache entry. That only holds while the client keeps the
		// connection open: a hang-up cancels the request, and cutBytes is then a lower bound.
		_ = http.NewResponseController(w).Flush()
		if cut, _ := io.Copy(io.Discard, s); cut > 0 {
			log.Warn(ctx, "Transcoded output is longer than the estimated length, the end of the file was cut",
				"id", id, "estimatedLength", limit, "cutBytes", cut)
		}
	}
	log.Trace(ctx, "Success sending transcoded file", "id", id, "size", c)
	return c, nil
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

// NewStream creates a non-seekable Stream from the given components.
func NewStream(mf *model.MediaFile, format string, bitRate int, r io.ReadCloser) *Stream {
	return &Stream{
		ctx:        context.Background(),
		mf:         mf,
		format:     format,
		bitRate:    bitRate,
		ReadCloser: r,
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

			release, err := job.ms.limiter.Acquire(ctx, limiterKey(ctx))
			if err != nil {
				log.Warn(ctx, "Refusing transcode: concurrent transcode limit reached",
					"id", job.mf.ID, "user", userName(ctx),
					"maxConcurrent", conf.Server.Transcoding.MaxConcurrent,
					"maxPerUser", conf.Server.Transcoding.MaxConcurrentPerUser)
				return nil, err
			}

			// Choose the context that drives the ffmpeg process.
			//
			// When the limiter is enabled, force the request context so a
			// client disconnect cancels ffmpeg and frees the slot promptly.
			// Otherwise a client could open many transcodes, disconnect
			// immediately, and still leave the configured cap's worth of
			// ffmpeg processes draining in the background — which is exactly
			// the DoS the limiter is meant to prevent.
			//
			// When the limiter is disabled, preserve the legacy behavior
			// governed by Transcoding.EnableCancellation so unchanged configs
			// keep their previous observable behavior.
			var transcodingCtx context.Context
			if job.ms.limiter.Enabled() || conf.Server.Transcoding.EnableCancellation {
				transcodingCtx = ctx
			} else {
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
				Duration:   job.mf.Duration,
			})
			if err != nil {
				release()
				log.Error(ctx, "Error starting transcoder", "id", job.mf.ID, err)
				return nil, os.ErrInvalid
			}
			// Tie the slot to the ffmpeg process: copyAndClose calls Close
			// on this reader after io.Copy returns, which is exactly when
			// ffmpeg has exited (either EOF or context cancellation).
			return &releasingReadCloser{ReadCloser: out, release: release}, nil
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

// limiterKey returns the per-user bucket key used by the transcode limiter.
// For anonymous requests (e.g. public shares) it returns the empty string,
// which signals the limiter to skip the per-user cap entirely — otherwise
// every anonymous viewer of a public share would collide on the same key
// and starve each other within MaxConcurrentPerUser slots. The global cap
// still applies and remains the protection against runaway anonymous load.
func limiterKey(ctx context.Context) string {
	if user, ok := request.UserFrom(ctx); ok {
		return user.UserName
	}
	return ""
}
