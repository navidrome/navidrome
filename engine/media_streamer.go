package engine

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/engine/ffmpeg"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
	"github.com/djherbis/fscache"
)

type MediaStreamer interface {
	NewStream(ctx context.Context, id string, maxBitRate int, format string) (*Stream, error)
}

func NewMediaStreamer(ds model.DataStore, ffm ffmpeg.FFmpeg, cache fscache.Cache) MediaStreamer {
	return &mediaStreamer{ds: ds, ffm: ffm, cache: cache}
}

type mediaStreamer struct {
	ds    model.DataStore
	ffm   ffmpeg.FFmpeg
	cache fscache.Cache
}

func (ms *mediaStreamer) NewStream(ctx context.Context, id string, maxBitRate int, reqFormat string) (*Stream, error) {
	mf, err := ms.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	bitRate, format := selectTranscodingOptions(mf, maxBitRate, reqFormat)
	s := &Stream{ctx: ctx, mf: mf, format: format, bitRate: bitRate}

	if format == "raw" {
		log.Debug(ctx, "Streaming raw file", "id", mf.ID, "path", mf.Path,
			"requestBitrate", maxBitRate, "requestFormat", reqFormat,
			"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix)
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

	key := cacheKey(id, bitRate, format)
	r, w, err := ms.cache.Get(key)
	if err != nil {
		log.Error(ctx, "Error creating stream caching buffer", "id", mf.ID, err)
		return nil, err
	}

	// If this is a brand new transcoding request, not in the cache, start transcoding
	if w != nil {
		log.Trace(ctx, "Cache miss. Starting new transcoding session", "id", mf.ID)
		out, err := ms.ffm.StartTranscoding(ctx, mf.Path, bitRate, format)
		if err != nil {
			log.Error(ctx, "Error starting transcoder", "id", mf.ID, err)
			return nil, os.ErrInvalid
		}
		go copyAndClose(ctx, w, out)()
	}

	// If it is in the cache, check if the stream is done being written. If so, return a ReaderSeeker
	if w == nil {
		size := getFinalCachedSize(r)
		if size > 0 {
			log.Debug(ctx, "Streaming cached file", "id", mf.ID, "path", mf.Path,
				"requestBitrate", maxBitRate, "requestFormat", reqFormat,
				"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix, "size", size)
			sr := io.NewSectionReader(r, 0, size)
			s.Reader = sr
			s.Closer = r
			s.Seeker = sr
			s.format = format
			return s, nil
		}
	}

	log.Debug(ctx, "Streaming transcoded file", "id", mf.ID, "path", mf.Path,
		"requestBitrate", maxBitRate, "requestFormat", reqFormat,
		"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix)
	// All other cases, just return a ReadCloser, without Seek capabilities
	s.Reader = r
	s.Closer = r
	s.format = format
	return s, nil
}

func copyAndClose(ctx context.Context, w io.WriteCloser, r io.ReadCloser) func() {
	return func() {
		_, err := io.Copy(w, r)
		if err != nil {
			log.Error(ctx, "Error copying data to cache", err)
		}
		err = r.Close()
		if err != nil {
			log.Error(ctx, "Error closing transcode output", err)
		}
		err = w.Close()
		if err != nil {
			log.Error(ctx, "Error closing cache", err)
		}
	}
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
func (s *Stream) Name() string        { return s.mf.Path }
func (s *Stream) ModTime() time.Time  { return s.mf.UpdatedAt }

func selectTranscodingOptions(mf *model.MediaFile, maxBitRate int, format string) (int, string) {
	var bitRate int

	if format == "raw" || !conf.Server.EnableDownsampling {
		return bitRate, "raw"
	} else {
		if maxBitRate == 0 {
			bitRate = mf.BitRate
		} else {
			bitRate = utils.MinInt(mf.BitRate, maxBitRate)
		}
		format = "mp3" //mf.Suffix
	}
	if conf.Server.MaxBitRate != 0 {
		bitRate = utils.MinInt(bitRate, conf.Server.MaxBitRate)
	}

	if bitRate == mf.BitRate {
		return bitRate, "raw"
	}
	return bitRate, format
}

func cacheKey(id string, bitRate int, format string) string {
	return fmt.Sprintf("%s.%d.%s", id, bitRate, format)
}

func getFinalCachedSize(r fscache.ReadAtCloser) int64 {
	cr, ok := r.(*fscache.CacheReader)
	if ok {
		size, final, err := cr.Size()
		if final && err == nil {
			return size
		}
	}
	return -1
}

func NewTranscodingCache() (fscache.Cache, error) {
	lru := fscache.NewLRUHaunter(0, conf.Server.MaxTranscodingCacheSize, 10*time.Minute)
	h := fscache.NewLRUHaunterStrategy(lru)
	cacheFolder := filepath.Join(conf.Server.DataFolder, consts.CacheDir)
	fs, err := fscache.NewFs(cacheFolder, 0755)
	if err != nil {
		return nil, err
	}
	return fscache.NewCacheWithHaunter(fs, h)
}
