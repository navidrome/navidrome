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
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

type FFmpeg interface {
	Transcode(ctx context.Context, command, path string, maxBitRate int) (io.ReadCloser, error)
	ExtractImage(ctx context.Context, path string) (io.ReadCloser, error)
	ConvertToWAV(ctx context.Context, path string) (io.ReadCloser, error)
	ConvertToFLAC(ctx context.Context, path string) (io.ReadCloser, error)
	Probe(ctx context.Context, files []string) (string, error)
	CmdPath() (string, error)
}

func New() FFmpeg {
	return &ffmpeg{}
}

const (
	extractImageCmd = "ffmpeg -i %s -an -vcodec copy -f image2pipe -"
	probeCmd        = "ffmpeg %s -f ffmetadata"
	createWavCmd    = "ffmpeg -i %s -c:a pcm_s16le -f wav -"
	createFLACCmd   = "ffmpeg -i %s -f flac -"
)

type ffmpeg struct{}

func (e *ffmpeg) Transcode(ctx context.Context, command, path string, maxBitRate int) (io.ReadCloser, error) {
	if _, err := ffmpegCmd(); err != nil {
		return nil, err
	}
	args := createFFmpegCommand(command, path, maxBitRate)
	return e.start(ctx, args)
}

func (e *ffmpeg) ExtractImage(ctx context.Context, path string) (io.ReadCloser, error) {
	if _, err := ffmpegCmd(); err != nil {
		return nil, err
	}
	args := createFFmpegCommand(extractImageCmd, path, 0)
	return e.start(ctx, args)
}

func (e *ffmpeg) ConvertToWAV(ctx context.Context, path string) (io.ReadCloser, error) {
	args := createFFmpegCommand(createWavCmd, path, 0)
	return e.start(ctx, args)
}

func (e *ffmpeg) ConvertToFLAC(ctx context.Context, path string) (io.ReadCloser, error) {
	args := createFFmpegCommand(createFLACCmd, path, 0)
	return e.start(ctx, args)
}

func (e *ffmpeg) Probe(ctx context.Context, files []string) (string, error) {
	if _, err := ffmpegCmd(); err != nil {
		return "", err
	}
	args := createProbeCommand(probeCmd, files)
	log.Trace(ctx, "Executing ffmpeg command", "args", args)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) // #nosec
	output, _ := cmd.CombinedOutput()
	return string(output), nil
}

func (e *ffmpeg) CmdPath() (string, error) {
	return ffmpegCmd()
}

func (e *ffmpeg) start(ctx context.Context, args []string) (io.ReadCloser, error) {
	log.Trace(ctx, "Executing ffmpeg command", "cmd", args)
	j := &ffCmd{args: args}
	j.PipeReader, j.out = io.Pipe()
	err := j.start()
	if err != nil {
		return nil, err
	}
	go j.wait()
	return j, nil
}

type ffCmd struct {
	*io.PipeReader
	out  *io.PipeWriter
	args []string
	cmd  *exec.Cmd
}

func (j *ffCmd) start() error {
	cmd := exec.Command(j.args[0], j.args[1:]...) // #nosec
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

func (j *ffCmd) wait() {
	if err := j.cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_ = j.out.CloseWithError(fmt.Errorf("%s exited with non-zero status code: %d", j.args[0], exitErr.ExitCode()))
		} else {
			_ = j.out.CloseWithError(fmt.Errorf("waiting %s cmd: %w", j.args[0], err))
		}
		return
	}
	_ = j.out.Close()
}

// Path will always be an absolute path
func createFFmpegCommand(cmd, path string, maxBitRate int) []string {
	split := strings.Split(fixCmd(cmd), " ")
	for i, s := range split {
		s = strings.ReplaceAll(s, "%s", path)
		s = strings.ReplaceAll(s, "%b", strconv.Itoa(maxBitRate))
		split[i] = s
	}

	return split
}

func createProbeCommand(cmd string, inputs []string) []string {
	split := strings.Split(fixCmd(cmd), " ")
	var args []string

	for _, s := range split {
		if s == "%s" {
			for _, inp := range inputs {
				args = append(args, "-i", inp)
			}
		} else {
			args = append(args, s)
		}
	}
	return args
}

func fixCmd(cmd string) string {
	split := strings.Split(cmd, " ")
	var result []string
	cmdPath, _ := ffmpegCmd()
	for _, s := range split {
		if s == "ffmpeg" || s == "ffmpeg.exe" {
			result = append(result, cmdPath)
		} else {
			result = append(result, s)
		}
	}
	return strings.Join(result, " ")
}

func ffmpegCmd() (string, error) {
	ffOnce.Do(func() {
		if conf.Server.FFmpegPath != "" {
			ffmpegPath = conf.Server.FFmpegPath
			ffmpegPath, ffmpegErr = exec.LookPath(ffmpegPath)
		} else {
			ffmpegPath, ffmpegErr = exec.LookPath("ffmpeg")
			if errors.Is(ffmpegErr, exec.ErrDot) {
				log.Trace("ffmpeg found in current folder '.'")
				ffmpegPath, ffmpegErr = exec.LookPath("./ffmpeg")
			}
		}
		if ffmpegErr == nil {
			log.Info("Found ffmpeg", "path", ffmpegPath)
			return
		}
	})
	return ffmpegPath, ffmpegErr
}

var (
	ffOnce     sync.Once
	ffmpegPath string
	ffmpegErr  error
)
