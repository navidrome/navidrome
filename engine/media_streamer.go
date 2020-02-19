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
)

type MediaStreamer interface {
	NewFileSystem(ctx context.Context, maxBitRate int, format string) (http.FileSystem, error)
}

func NewMediaStreamer(ds model.DataStore, ffm ffmpeg.FFmpeg) MediaStreamer {
	return &mediaStreamer{ds: ds, ffm: ffm}
}

type mediaStreamer struct {
	ds  model.DataStore
	ffm ffmpeg.FFmpeg
}

func (ms *mediaStreamer) NewFileSystem(ctx context.Context, maxBitRate int, format string) (http.FileSystem, error) {
	cacheFolder := filepath.Join(conf.Server.DataFolder, consts.CacheDir)
	err := os.MkdirAll(cacheFolder, 0755)
	if err != nil {
		log.Error("Could not create cache folder", "folder", cacheFolder, err)
		return nil, err
	}
	return &mediaFileSystem{ctx: ctx, ds: ms.ds, ffm: ms.ffm, maxBitRate: maxBitRate, format: format, cacheFolder: cacheFolder}, nil
}

type mediaFileSystem struct {
	ctx         context.Context
	ds          model.DataStore
	maxBitRate  int
	format      string
	cacheFolder string
	ffm         ffmpeg.FFmpeg
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

	cachedFile := fs.cacheFilePath(mf, bitRate, format)
	if _, err := os.Stat(cachedFile); !os.IsNotExist(err) {
		log.Debug(fs.ctx, "Streaming cached transcoded", "id", mf.ID, "path", mf.Path,
			"requestBitrate", bitRate, "requestFormat", format,
			"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix)
		return os.Open(cachedFile)
	}

	log.Debug(fs.ctx, "Streaming transcoded file", "id", mf.ID, "path", mf.Path,
		"requestBitrate", bitRate, "requestFormat", format,
		"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix)

	return fs.transcodeFile(mf, bitRate, format, cachedFile)
}

func (fs *mediaFileSystem) cacheFilePath(mf *model.MediaFile, bitRate int, format string) string {
	// Break the cache in subfolders, to avoid too many files in the same folder
	subDir := strings.ToLower(mf.ID[:2])
	subDir = filepath.Join(fs.cacheFolder, subDir)
	// Make sure the subfolder to exist
	os.Mkdir(subDir, 0755)
	return filepath.Join(subDir, fmt.Sprintf("%s.%d.%s", mf.ID, bitRate, format))
}

func (fs *mediaFileSystem) transcodeFile(mf *model.MediaFile, bitRate int, format, cacheFile string) (*transcodingFile, error) {
	out, err := fs.ffm.StartTranscoding(fs.ctx, mf.Path, bitRate, format)
	if err != nil {
		log.Error("Error starting transcoder", "id", mf.ID, err)
		return nil, os.ErrInvalid
	}
	buf, err := newStreamBuffer(cacheFile)
	if err != nil {
		log.Error("Error creating stream buffer", "id", mf.ID, err)
		return nil, os.ErrInvalid
	}
	r, err := buf.NewReader()
	if err != nil {
		log.Error("Error opening stream reader", "id", mf.ID, err)
		return nil, os.ErrInvalid
	}
	go func() {
		io.Copy(buf, out)
		out.Close()
		buf.Sync()
		buf.Close()
	}()
	s := &transcodingFile{
		ctx:     fs.ctx,
		mf:      mf,
		bitRate: bitRate,
	}
	s.File = r
	return s, nil
}

type transcodingFile struct {
	ctx     context.Context
	mf      *model.MediaFile
	bitRate int
	http.File
}

func (h *transcodingFile) Stat() (os.FileInfo, error) {
	return &streamHandlerFileInfo{mf: h.mf, bitRate: h.bitRate}, nil
}

// Don't return EOF, just wait for more data. When the request ends, this "File" will be closed, and then
// the Read will be interrupted
func (h *transcodingFile) Read(b []byte) (int, error) {
	for {
		n, err := h.File.Read(b)
		if n > 0 {
			return n, nil
		} else if err != io.EOF {
			return n, err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

type streamHandlerFileInfo struct {
	mf      *model.MediaFile
	bitRate int
}

func (f *streamHandlerFileInfo) Name() string       { return f.mf.Title }
func (f *streamHandlerFileInfo) Size() int64        { return int64((f.mf.Duration)*f.bitRate*1000) / 8 }
func (f *streamHandlerFileInfo) Mode() os.FileMode  { return os.FileMode(0777) }
func (f *streamHandlerFileInfo) ModTime() time.Time { return f.mf.UpdatedAt }
func (f *streamHandlerFileInfo) IsDir() bool        { return false }
func (f *streamHandlerFileInfo) Sys() interface{}   { return nil }

// From: https://stackoverflow.com/a/44322300
type streamBuffer struct {
	*os.File
}

func (mb *streamBuffer) NewReader() (http.File, error) {
	f, err := os.Open(mb.Name())
	if err != nil {
		return nil, err
	}
	return f, nil
}

func newStreamBuffer(name string) (*streamBuffer, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	return &streamBuffer{File: f}, nil
}
