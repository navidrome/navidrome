package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/engine/ffmpeg"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
	"gopkg.in/djherbis/fscache.v0"
)

type MediaStreamer interface {
	NewFileSystem(ctx context.Context, maxBitRate int, format string) (http.FileSystem, error)
}

func NewMediaStreamer(ds model.DataStore, ffm ffmpeg.FFmpeg, cache fscache.Cache) MediaStreamer {
	return &mediaStreamer{ds: ds, ffm: ffm, cache: cache}
}

type mediaStreamer struct {
	ds    model.DataStore
	ffm   ffmpeg.FFmpeg
	cache fscache.Cache
}

func (ms *mediaStreamer) NewFileSystem(ctx context.Context, maxBitRate int, format string) (http.FileSystem, error) {
	return &mediaFileSystem{ctx: ctx, ds: ms.ds, ffm: ms.ffm, cache: ms.cache, maxBitRate: maxBitRate, format: format}, nil
}

type mediaFileSystem struct {
	ctx        context.Context
	ds         model.DataStore
	maxBitRate int
	format     string
	ffm        ffmpeg.FFmpeg
	cache      fscache.Cache
}

func (fs *mediaFileSystem) selectTranscodingOptions(mf *model.MediaFile) (string, int) {
	var bitRate int
	var format string

	if fs.format == "raw" || !conf.Server.EnableDownsampling {
		return "raw", bitRate
	} else {
		if fs.maxBitRate == 0 {
			bitRate = mf.BitRate
		} else {
			bitRate = utils.MinInt(mf.BitRate, fs.maxBitRate)
		}
		format = "mp3" //mf.Suffix
	}
	if conf.Server.MaxBitRate != 0 {
		bitRate = utils.MinInt(bitRate, conf.Server.MaxBitRate)
	}

	if bitRate == mf.BitRate {
		return "raw", bitRate
	}
	return format, bitRate
}

func (fs *mediaFileSystem) Open(name string) (http.File, error) {
	id := strings.Trim(name, "/")
	mf, err := fs.ds.MediaFile(fs.ctx).Get(id)
	if err == model.ErrNotFound {
		return nil, os.ErrNotExist
	}
	if err != nil {
		log.Error("Error opening mediaFile", "id", id, err)
		return nil, os.ErrInvalid
	}

	format, bitRate := fs.selectTranscodingOptions(mf)
	if format == "raw" {
		log.Debug(fs.ctx, "Streaming raw file", "id", mf.ID, "path", mf.Path,
			"requestBitrate", bitRate, "requestFormat", format,
			"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix)
		return os.Open(mf.Path)
	}

	log.Debug(fs.ctx, "Streaming transcoded file", "id", mf.ID, "path", mf.Path,
		"requestBitrate", bitRate, "requestFormat", format,
		"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix)

	return fs.transcodeFile(mf, bitRate, format)
}

func (fs *mediaFileSystem) transcodeFile(mf *model.MediaFile, bitRate int, format string) (*transcodingFile, error) {
	key := fmt.Sprintf("%s.%d.%s", mf.ID, bitRate, format)
	r, w, err := fs.cache.Get(key)
	if err != nil {
		log.Error("Error creating stream caching buffer", "id", mf.ID, err)
		return nil, os.ErrInvalid
	}

	// If it is a new file (not found in the cached), start a new transcoding session
	if w != nil {
		log.Debug("File not found in cache. Starting new transcoding session", "id", mf.ID)
		out, err := fs.ffm.StartTranscoding(fs.ctx, mf.Path, bitRate, format)
		if err != nil {
			log.Error("Error starting transcoder", "id", mf.ID, err)
			return nil, os.ErrInvalid
		}
		go func() {
			io.Copy(w, out)
			out.Close()
			w.Close()
		}()
	} else {
		log.Debug("Reading transcoded file from cache", "id", mf.ID)
	}

	return newTranscodingFile(fs.ctx, r, mf, bitRate), nil
}

// transcodingFile Implements http.File interface, required for the FileSystem. It needs a Closer, a Reader and
// a Seeker for the same stream. Because the fscache package only provides a ReaderAtCloser (without the Seek()
// method), we wrap that reader with a SectionReader, which provides a Seek(). But we still need the original
// reader, as we need to close the stream when the transfer is complete
func newTranscodingFile(ctx context.Context, reader fscache.ReadAtCloser,
	mf *model.MediaFile, bitRate int) *transcodingFile {

	size := int64(mf.Duration*float32(bitRate*1000)) / 8
	return &transcodingFile{
		ctx:        ctx,
		mf:         mf,
		bitRate:    bitRate,
		size:       size,
		closer:     reader,
		ReadSeeker: io.NewSectionReader(reader, 0, size),
	}
}

type transcodingFile struct {
	ctx     context.Context
	mf      *model.MediaFile
	bitRate int
	size    int64
	closer  io.Closer
	io.ReadSeeker
}

func (tf *transcodingFile) Stat() (os.FileInfo, error) {
	return &streamHandlerFileInfo{f: tf}, nil
}

func (tf *transcodingFile) Close() error {
	return tf.closer.Close()
}

func (tf *transcodingFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

type streamHandlerFileInfo struct {
	f *transcodingFile
}

func (fi *streamHandlerFileInfo) Name() string       { return fi.f.mf.Title }
func (fi *streamHandlerFileInfo) ModTime() time.Time { return fi.f.mf.UpdatedAt }
func (fi *streamHandlerFileInfo) Size() int64        { return fi.f.size }
func (fi *streamHandlerFileInfo) Mode() os.FileMode  { return os.FileMode(0777) }
func (fi *streamHandlerFileInfo) IsDir() bool        { return false }
func (fi *streamHandlerFileInfo) Sys() interface{}   { return nil }

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
