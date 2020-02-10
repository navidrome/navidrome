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
	Duration() int
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

func (m *rawMediaStream) Duration() int {
	return m.mf.Duration
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
	pos     int64
}

func (m *transcodedMediaStream) Read(p []byte) (n int, err error) {
	// Open the pipe and optionally skip a initial chunk of the stream (to simulate a Seek)
	if m.pipe == nil {
		m.pipe, err = newTranscode(m.ctx, m.mf.Path, m.bitRate, m.format)
		if err != nil {
			return 0, err
		}
		if m.skip > 0 {
			_, err := io.CopyN(ioutil.Discard, m.pipe, m.skip)
			m.pos = m.skip
			if err != nil {
				return 0, err
			}
		}
	}
	n, err = m.pipe.Read(p)
	m.pos += int64(n)
	if err == io.EOF {
		m.Close()
	}
	return
}

// This is an attempt to make a pipe seekable. It is very wasteful, restarting the stream every time
// a Seek happens. This is ok-ish for audio, but would kill the server for video.
func (m *transcodedMediaStream) Seek(offset int64, whence int) (int64, error) {
	size := int64((m.mf.Duration)*m.bitRate*1000) / 8
	log.Trace(m.ctx, "Seeking transcoded stream", "path", m.mf.Path, "offset", offset, "whence", whence, "size", size)

	switch whence {
	case io.SeekEnd:
		m.skip = size - offset
		offset = size
	case io.SeekStart:
		m.skip = offset
	case io.SeekCurrent:
		io.CopyN(ioutil.Discard, m.pipe, offset)
		m.pos += offset
		offset = m.pos
	}

	// If need to Seek to a previous position, close the pipe (will be restarted on next Read)
	var err error
	if whence != io.SeekCurrent {
		if m.pipe != nil {
			err = m.Close()
		}
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

func (m *transcodedMediaStream) Duration() int {
	return m.mf.Duration
}

func (m *transcodedMediaStream) Close() error {
	log.Trace(m.ctx, "Closing stream", "id", m.mf.ID, "path", m.mf.Path)
	err := m.pipe.Close()
	m.pipe = nil
	m.pos = 0
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
