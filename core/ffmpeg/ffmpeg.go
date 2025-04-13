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
	Transcode(ctx context.Context, command, path string, maxBitRate, offset int) (io.ReadCloser, error)
	ExtractImage(ctx context.Context, path string) (io.ReadCloser, error)
	Probe(ctx context.Context, files []string) (string, error)
	CmdPath() (string, error)
	IsAvailable() bool
	Version() string
}

func New() FFmpeg {
	return &ffmpeg{}
}

const (
	extractImageCmd = "ffmpeg -i %s -map 0:v -map -0:V -vcodec copy -f image2pipe -"
	probeCmd        = "ffmpeg %s -f ffmetadata"
)

type ffmpeg struct{}

func (e *ffmpeg) Transcode(ctx context.Context, command, path string, maxBitRate, offset int) (io.ReadCloser, error) {
	if _, err := ffmpegCmd(); err != nil {
		return nil, err
	}
	// First make sure the file exists
	if err := fileExists(path); err != nil {
		return nil, err
	}
	args := createFFmpegCommand(command, path, maxBitRate, offset)
	return e.start(ctx, args)
}

func (e *ffmpeg) ExtractImage(ctx context.Context, path string) (io.ReadCloser, error) {
	if _, err := ffmpegCmd(); err != nil {
		return nil, err
	}
	// First make sure the file exists
	if err := fileExists(path); err != nil {
		return nil, err
	}
	args := createFFmpegCommand(extractImageCmd, path, 0, 0)
	return e.start(ctx, args)
}

func fileExists(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory", path)
	}
	return nil
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

func (e *ffmpeg) IsAvailable() bool {
	_, err := ffmpegCmd()
	return err == nil
}

// Version executes ffmpeg -version and extracts the version from the output.
// Sample output: ffmpeg version 6.0 Copyright (c) 2000-2023 the FFmpeg developers
func (e *ffmpeg) Version() string {
	cmd, err := ffmpegCmd()
	if err != nil {
		return "N/A"
	}
	out, err := exec.Command(cmd, "-version").CombinedOutput() // #nosec
	if err != nil {
		return "N/A"
	}
	parts := strings.Split(string(out), " ")
	if len(parts) < 3 {
		return "N/A"
	}
	return parts[2]
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
	if log.IsGreaterOrEqualTo(log.LevelTrace) {
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
func createFFmpegCommand(cmd, path string, maxBitRate, offset int) []string {
	var args []string
	for _, s := range fixCmd(cmd) {
		if strings.Contains(s, "%s") {
			s = strings.ReplaceAll(s, "%s", path)
			args = append(args, s)
			if offset > 0 && !strings.Contains(cmd, "%t") {
				args = append(args, "-ss", strconv.Itoa(offset))
			}
		} else {
			s = strings.ReplaceAll(s, "%t", strconv.Itoa(offset))
			s = strings.ReplaceAll(s, "%b", strconv.Itoa(maxBitRate))
			args = append(args, s)
		}
	}
	return args
}

func createProbeCommand(cmd string, inputs []string) []string {
	var args []string
	for _, s := range fixCmd(cmd) {
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

func fixCmd(cmd string) []string {
	split := strings.Fields(cmd)
	cmdPath, _ := ffmpegCmd()
	for i, s := range split {
		if s == "ffmpeg" || s == "ffmpeg.exe" {
			split[i] = cmdPath
		}
	}
	return split
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

// These variables are accessible here for tests. Do not use them directly in production code. Use ffmpegCmd() instead.
var (
	ffOnce     sync.Once
	ffmpegPath string
	ffmpegErr  error
)
