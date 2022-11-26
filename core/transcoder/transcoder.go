package transcoder

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
)

type Transcoder interface {
	Start(ctx context.Context, command, path string, maxBitRate int) (f io.ReadCloser, err error)
}

func New() Transcoder {
	return &externalTranscoder{}
}

type externalTranscoder struct{}

func (e *externalTranscoder) Start(ctx context.Context, command, path string, maxBitRate int) (f io.ReadCloser, err error) {
	args := createTranscodeCommand(command, path, maxBitRate)

	log.Trace(ctx, "Executing transcoding command", "cmd", args)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) // #nosec
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
		s = strings.ReplaceAll(s, "%s", path)
		s = strings.ReplaceAll(s, "%b", strconv.Itoa(maxBitRate))
		split[i] = s
	}

	return split
}
