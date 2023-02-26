package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"golang.org/x/exp/slices"
)

type FFmpeg interface {
	Transcode(ctx context.Context, command, path string, maxBitRate int) (io.ReadCloser, error)
	ExtractImage(ctx context.Context, path string) (io.ReadCloser, error)
	Probe(ctx context.Context, files []string) (string, error)
	CmdPath() (string, error)
}

func New() FFmpeg {
	return &ffmpeg{}
}

const (
	extractImageCmd = "ffmpeg -i %s -an -c:v %i -f image2pipe -"
	probeCmd        = "ffmpeg %s -f ffmetadata"
)

var (
	//   Stream #0:0: Video: theora, yuv444p, 214x152 [SAR 1:1 DAR 107:76], 25 tbr, 25 tbn, 25 tbc
	//   Stream #0:0: Video: webp, yuv420p
	//   Stream #0:0: Video: mjpeg (Baseline),
	//   Stream #0:0: Video: png, rgb24(pc),
	videoStreamTypeRegex   = regexp.MustCompile(`Stream #\d+:\d*(?:\(\w+\))?: Video: ([^\s,]+)`)
	supportedRawStreamType = []string{"webp", "mjpeg", "png"}
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

	imgFormat, err := e.GetVideoStreamType(ctx, path)
	if err != nil {
		return nil, err
	}

	args := createCoverExtractCommand(imgFormat, path)
	return e.start(ctx, args)
}

func (e *ffmpeg) GetVideoStreamType(ctx context.Context, file_path string) (string, error) {
	output, err := e.Probe(ctx, []string{file_path})
	if err == nil {
		return "", nil
	}

	return matchProbedVideoStreamType(output)
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

func createCoverExtractCommand(imgFmt, audioPath string) []string {
	ffmpegOutputFormat := "mjpeg"
	if slices.Contains(supportedRawStreamType, imgFmt) {
		ffmpegOutputFormat = "copy"
	}

	cmd := strings.ReplaceAll(extractImageCmd, "%i", ffmpegOutputFormat)
	return createFFmpegCommand(cmd, audioPath, 0)
}

func matchProbedVideoStreamType(output string) (string, error) {
	match := videoStreamTypeRegex.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("could not detect stream cover type")
	}

	return match[1], nil
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
