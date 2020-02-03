package engine

import (
	"context"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
)

type MediaStreamer interface {
	NewStream(ctx context.Context, id string, maxBitRate int, format string) (mediaStream, error)
}

func NewMediaStreamer(ds model.DataStore) MediaStreamer {
	return &mediaStreamer{ds: ds}
}

type mediaStream interface {
	io.ReadSeeker
	ContentType() string
	Name() string
	ModTime() time.Time
	Close() error
}

type mediaStreamer struct {
	ds model.DataStore
}

func (ms *mediaStreamer) NewStream(ctx context.Context, id string, maxBitRate int, format string) (mediaStream, error) {
	mf, err := ms.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	var bitRate int

	if format == "raw" || !conf.Server.EnableDownsampling {
		bitRate = mf.BitRate
		format = mf.Suffix
	} else {
		if maxBitRate == 0 {
			bitRate = mf.BitRate
		} else {
			bitRate = utils.MinInt(mf.BitRate, maxBitRate)
		}
		format = mf.Suffix
	}
	if conf.Server.MaxBitRate != 0 {
		bitRate = utils.MinInt(bitRate, conf.Server.MaxBitRate)
	}

	var stream mediaStream

	if bitRate == mf.BitRate && mime.TypeByExtension("."+format) == mf.ContentType() {
		log.Debug(ctx, "Streaming raw file", "id", mf.ID, "path", mf.Path,
			"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix)

		f, err := os.Open(mf.Path)
		if err != nil {
			return nil, err
		}
		stream = &rawMediaStream{ctx: ctx, mf: mf, file: f}
		return stream, nil
	}

	log.Debug(ctx, "Streaming transcoded file", "id", mf.ID, "path", mf.Path,
		"requestBitrate", bitRate, "requestFormat", format,
		"originalBitrate", mf.BitRate, "originalFormat", mf.Suffix)

	f := &transcodedMediaStream{ctx: ctx, mf: mf, bitRate: bitRate, format: format}
	return f, err
}

type rawMediaStream struct {
	file *os.File
	ctx  context.Context
	mf   *model.MediaFile
}

func (m *rawMediaStream) Read(p []byte) (n int, err error) {
	return m.file.Read(p)
}

func (m *rawMediaStream) Seek(offset int64, whence int) (int64, error) {
	return m.file.Seek(offset, whence)
}

func (m *rawMediaStream) ContentType() string {
	return m.mf.ContentType()
}

func (m *rawMediaStream) Name() string {
	return m.mf.Path
}

func (m *rawMediaStream) ModTime() time.Time {
	return m.mf.UpdatedAt
}

func (m *rawMediaStream) Close() error {
	log.Trace(m.ctx, "Closing file", "id", m.mf.ID, "path", m.mf.Path)
	return m.file.Close()
}

type transcodedMediaStream struct {
	ctx     context.Context
	mf      *model.MediaFile
	pipe    io.ReadCloser
	bitRate int
	format  string
	skip    int64
}

func (m *transcodedMediaStream) Read(p []byte) (n int, err error) {
	if m.pipe == nil {
		m.pipe, err = newTranscode(m.ctx, m.mf.Path, m.bitRate, m.format)
		if err != nil {
			return 0, err
		}
		if m.skip > 0 {
			_, err := io.CopyN(ioutil.Discard, m.pipe, m.skip)
			if err != nil {
				return 0, err
			}
		}
	}
	n, err = m.pipe.Read(p)
	if err == io.EOF {
		m.Close()
	}
	return
}

// This Seek function assumes internal details of http.ServeContent's implementation
// A better approach would be to implement a http.FileSystem and use http.FileServer
func (m *transcodedMediaStream) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekEnd {
		if offset == 0 {
			size := (m.mf.Duration) * m.bitRate * 1000
			return int64(size / 8), nil
		}
		panic("seeking stream backwards not supported")
	}
	m.skip = offset
	var err error
	if m.pipe != nil {
		err = m.Close()
	}
	return offset, err
}

func (m *transcodedMediaStream) ContentType() string {
	return mime.TypeByExtension(".mp3")
}

func (m *transcodedMediaStream) Name() string {
	return m.mf.Path
}

func (m *transcodedMediaStream) ModTime() time.Time {
	return m.mf.UpdatedAt
}

func (m *transcodedMediaStream) Close() error {
	log.Trace(m.ctx, "Closing stream", "id", m.mf.ID, "path", m.mf.Path)
	err := m.pipe.Close()
	m.pipe = nil
	return err
}

func newTranscode(ctx context.Context, path string, maxBitRate int, format string) (f io.ReadCloser, err error) {
	cmdLine, args := createTranscodeCommand(path, maxBitRate, format)

	log.Trace(ctx, "Executing ffmpeg command", "arg0", cmdLine, "args", args)
	cmd := exec.Command(cmdLine, args...)
	cmd.Stderr = os.Stderr
	if f, err = cmd.StdoutPipe(); err != nil {
		return f, err
	}
	return f, cmd.Start()
}

func createTranscodeCommand(path string, maxBitRate int, format string) (string, []string) {
	cmd := conf.Server.DownsampleCommand

	split := strings.Split(cmd, " ")
	for i, s := range split {
		s = strings.Replace(s, "%s", path, -1)
		s = strings.Replace(s, "%b", strconv.Itoa(maxBitRate), -1)
		split[i] = s
	}

	return split[0], split[1:]
}
