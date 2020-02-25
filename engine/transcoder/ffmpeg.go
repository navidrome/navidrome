package transcoder

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
)

type Transcoder interface {
	Start(ctx context.Context, path string, maxBitRate int, format string) (f io.ReadCloser, err error)
}

func New() Transcoder {
	return &ffmpeg{}
}

type ffmpeg struct{}

func (ff *ffmpeg) Start(ctx context.Context, path string, maxBitRate int, format string) (f io.ReadCloser, err error) {
	arg0, args := createTranscodeCommand(path, maxBitRate, format)

	log.Trace(ctx, "Executing ffmpeg command", "cmd", arg0, "args", args)
	cmd := exec.Command(arg0, args...)
	cmd.Stderr = os.Stderr
	if f, err = cmd.StdoutPipe(); err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}
	go cmd.Wait() // prevent zombies
	return
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
