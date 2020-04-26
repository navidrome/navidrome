package transcoder

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/deluan/navidrome/log"
)

type Transcoder interface {
	Start(ctx context.Context, command, path string, maxBitRate int) (f io.ReadCloser, err error)
}

func New() Transcoder {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Error("Unable to find ffmpeg", err)
	}
	log.Debug("Found ffmpeg", "path", path)
	return &ffmpeg{}
}

type ffmpeg struct{}

func (ff *ffmpeg) Start(ctx context.Context, command, path string, maxBitRate int) (f io.ReadCloser, err error) {
	args := createTranscodeCommand(command, path, maxBitRate)

	log.Trace(ctx, "Executing ffmpeg command", "cmd", args)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	if f, err = cmd.StdoutPipe(); err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}

	go func() { _ = cmd.Wait() }() // prevent zombies

	return
}

// Path will always be an absolute path
func createTranscodeCommand(cmd, path string, maxBitRate int) []string {
	split := strings.Split(cmd, " ")
	for i, s := range split {
		s = strings.Replace(s, "%s", path, -1)
		s = strings.Replace(s, "%b", strconv.Itoa(maxBitRate), -1)
		split[i] = s
	}

	return split
}
