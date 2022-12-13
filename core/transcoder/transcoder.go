package transcoder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
)

type Transcoder interface {
	Start(ctx context.Context, command, path string, maxBitRate int) (io.ReadCloser, error)
}

func New() Transcoder {
	return &externalTranscoder{}
}

type externalTranscoder struct{}

func (e *externalTranscoder) Start(ctx context.Context, command, path string, maxBitRate int) (io.ReadCloser, error) {
	args := createTranscodeCommand(command, path, maxBitRate)
	log.Trace(ctx, "Executing transcoding command", "cmd", args)
	j := &Cmd{ctx: ctx, args: args}
	j.PipeReader, j.out = io.Pipe()
	err := j.start()
	if err != nil {
		return nil, err
	}
	go j.wait()
	return j, nil
}

type Cmd struct {
	*io.PipeReader
	out  *io.PipeWriter
	ctx  context.Context
	args []string
	cmd  *exec.Cmd
}

func (j *Cmd) start() error {
	cmd := exec.CommandContext(j.ctx, j.args[0], j.args[1:]...) // #nosec
	cmd.Stdout = j.out
	cmd.Stderr = os.Stderr
	j.cmd = cmd

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting cmd: %w", err)
	}
	return nil
}

func (j *Cmd) wait() {
	var exitErr *exec.ExitError
	if err := j.cmd.Wait(); err != nil && !errors.As(err, &exitErr) {
		_ = j.out.CloseWithError(fmt.Errorf("waiting cmd: %w", err))
		return
	}
	if code := j.cmd.ProcessState.ExitCode(); code > 1 {
		_ = j.out.CloseWithError(fmt.Errorf("%s exited with non-zero status code: %d", j.args[0], code))
		return
	}
	if j.ctx.Err() != nil {
		_ = j.out.CloseWithError(j.ctx.Err())
		return
	}
	_ = j.out.Close()
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
