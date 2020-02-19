package ffmpeg

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

type FFmpeg interface {
	StartTranscoding(ctx context.Context, path string, maxBitRate int, format string) (f io.ReadCloser, err error)
}

func New() FFmpeg {
	return &ffmpeg{}
}

type ffmpeg struct{}

func (ff *ffmpeg) StartTranscoding(ctx context.Context, path string, maxBitRate int, format string) (f io.ReadCloser, err error) {
	cmdLine, args := createTranscodeCommand(path, maxBitRate, format)

	log.Trace(ctx, "Executing ffmpeg command", "arg0", cmdLine, "args", args)
	cmd := exec.Command(cmdLine, args...)
	cmd.Stderr = os.Stderr
	if f, err = cmd.StdoutPipe(); err != nil {
		return f, err
	}
	if err = cmd.Start(); err != nil {
		return f, err
	}
	go cmd.Wait() // prevent zombies
	return f, err
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
