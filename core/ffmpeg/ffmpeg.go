package ffmpeg

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

type FFmpeg interface {
	Transcode(ctx context.Context, command, path string, maxBitRate int) (io.ReadCloser, error)
	ExtractImage(ctx context.Context, path string) (io.ReadCloser, error)
	// TODO Move scanner ffmpeg probe to here
}

func New() FFmpeg {
	return &ffmpeg{}
}

const extractImageCmd = "ffmpeg -i %s -an -vcodec copy -f image2pipe -"

type ffmpeg struct{}

func (e *ffmpeg) Transcode(ctx context.Context, command, path string, maxBitRate int) (io.ReadCloser, error) {
	args := createFFmpegCommand(command, path, maxBitRate)
	return e.start(ctx, args)
}

func (e *ffmpeg) ExtractImage(ctx context.Context, path string) (io.ReadCloser, error) {
	args := createFFmpegCommand(extractImageCmd, path, 0)
	return e.start(ctx, args)
}

func (e *ffmpeg) start(ctx context.Context, args []string) (io.ReadCloser, error) {
	log.Trace(ctx, "Executing ffmpeg command", "cmd", args)
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
	if log.CurrentLevel() >= log.LevelTrace {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = io.Discard
	}
	j.cmd = cmd

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting cmd: %w", err)
	}
	return nil
}

func (j *Cmd) wait() {
	if err := j.cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_ = j.out.CloseWithError(fmt.Errorf("%s exited with non-zero status code: %d", j.args[0], exitErr.ExitCode()))
		} else {
			_ = j.out.CloseWithError(fmt.Errorf("waiting %s cmd: %w", j.args[0], err))
		}
		return
	}
	if j.ctx.Err() != nil {
		_ = j.out.CloseWithError(j.ctx.Err())
		return
	}
	_ = j.out.Close()
}

// Path will always be an absolute path
func createFFmpegCommand(cmd, path string, maxBitRate int) []string {
	split := strings.Split(cmd, " ")
	for i, s := range split {
		s = strings.ReplaceAll(s, "%s", path)
		s = strings.ReplaceAll(s, "%b", strconv.Itoa(maxBitRate))
		split[i] = s
	}

	return split
}
