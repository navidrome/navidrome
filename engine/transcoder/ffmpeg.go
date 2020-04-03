package transcoder

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/deluan/navidrome/log"
)

type Transcoder interface {
	Start(ctx context.Context, command, path string, maxBitRate int, format string) (f io.ReadCloser, err error)
}

func New() Transcoder {
	return &ffmpeg{}
}

type ffmpeg struct{}

func (ff *ffmpeg) Start(ctx context.Context, command, path string, maxBitRate int, format string) (f io.ReadCloser, err error) {
	arg0, args := createTranscodeCommand(command, path, maxBitRate, format)

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

func createTranscodeCommand(cmd, path string, maxBitRate int, format string) (string, []string) {
	args := strings.Split(cmd, " ")
	for i, s := range args {
		if fpath, err := filepath.Abs(path); err != nil{
			s = strings.Replace(s, "%s", fpath, -1)
		}
		s = strings.Replace(s, "%b", strconv.Itoa(maxBitRate), -1)
		args[i] = s
	}

	return args[0], args[1:]
}
