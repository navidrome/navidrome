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

// TranscodeOptions contains all parameters for a transcoding operation.
type TranscodeOptions struct {
	Command    string // DB command template (used to detect custom vs default)
	Format     string // Target format (mp3, opus, aac, flac)
	FilePath   string
	BitRate    int // kbps, 0 = codec default
	SampleRate int // 0 = no constraint
	Channels   int // 0 = no constraint
	BitDepth   int // 0 = no constraint; valid values: 16, 24, 32
	Offset     int // seconds
}

type FFmpeg interface {
	Transcode(ctx context.Context, opts TranscodeOptions) (io.ReadCloser, error)
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

func (e *ffmpeg) Transcode(ctx context.Context, opts TranscodeOptions) (io.ReadCloser, error) {
	if _, err := ffmpegCmd(); err != nil {
		return nil, err
	}
	if err := fileExists(opts.FilePath); err != nil {
		return nil, err
	}
	var args []string
	if isDefaultCommand(opts.Format, opts.Command) {
		args = buildDynamicArgs(opts)
	} else {
		args = buildTemplateArgs(opts)
	}
	return e.start(ctx, args)
}

func (e *ffmpeg) ExtractImage(ctx context.Context, path string) (io.ReadCloser, error) {
	if _, err := ffmpegCmd(); err != nil {
		return nil, err
	}
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
	err := j.start(ctx)
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

func (j *ffCmd) start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, j.args[0], j.args[1:]...) // #nosec
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

// defaultCommands maps format to the known default command template.
// Used to detect whether a user has customized their transcoding command.
var defaultCommands = map[string]string{
	"mp3":  "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -f mp3 -",
	"opus": "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a libopus -f opus -",
	"aac":  "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -",
	"flac": "ffmpeg -i %s -ss %t -map 0:a:0 -v 0 -c:a flac -f flac -",
}

// formatCodecMap maps target format to ffmpeg codec flag.
var formatCodecMap = map[string]string{
	"mp3":  "libmp3lame",
	"opus": "libopus",
	"aac":  "aac",
	"flac": "flac",
}

// formatOutputMap maps target format to ffmpeg output format flag (-f).
var formatOutputMap = map[string]string{
	"mp3":  "mp3",
	"opus": "opus",
	"aac":  "adts",
	"flac": "flac",
}

// isDefaultCommand returns true if the command matches the known default for this format.
func isDefaultCommand(format, command string) bool {
	defaultCmd, ok := defaultCommands[format]
	return ok && command == defaultCmd
}

// buildDynamicArgs programmatically constructs ffmpeg arguments for known formats,
// including all transcoding parameters (bitrate, sample rate, channels).
func buildDynamicArgs(opts TranscodeOptions) []string {
	cmdPath, _ := ffmpegCmd()
	args := []string{cmdPath, "-i", opts.FilePath}

	if opts.Offset > 0 {
		args = append(args, "-ss", strconv.Itoa(opts.Offset))
	}

	args = append(args, "-map", "0:a:0")

	if codec, ok := formatCodecMap[opts.Format]; ok {
		args = append(args, "-c:a", codec)
	}

	if opts.BitRate > 0 {
		args = append(args, "-b:a", strconv.Itoa(opts.BitRate)+"k")
	}
	if opts.SampleRate > 0 {
		args = append(args, "-ar", strconv.Itoa(opts.SampleRate))
	}
	if opts.Channels > 0 {
		args = append(args, "-ac", strconv.Itoa(opts.Channels))
	}
	// Only pass -sample_fmt for lossless output formats where bit depth matters.
	// Lossy codecs (mp3, aac, opus) handle sample format conversion internally,
	// and passing interleaved formats like "s16" causes silent failures.
	if opts.BitDepth >= 16 && isLosslessOutputFormat(opts.Format) {
		args = append(args, "-sample_fmt", bitDepthToSampleFmt(opts.BitDepth))
	}

	args = append(args, "-v", "0")

	if outputFmt, ok := formatOutputMap[opts.Format]; ok {
		args = append(args, "-f", outputFmt)
	}

	args = append(args, "-")
	return args
}

// buildTemplateArgs handles user-customized command templates, with dynamic injection
// of sample rate and channels when the template doesn't already include them.
func buildTemplateArgs(opts TranscodeOptions) []string {
	args := createFFmpegCommand(opts.Command, opts.FilePath, opts.BitRate, opts.Offset)

	// Dynamically inject -ar, -ac, and -sample_fmt for custom templates that don't include them
	if opts.SampleRate > 0 {
		args = injectBeforeOutput(args, "-ar", strconv.Itoa(opts.SampleRate))
	}
	if opts.Channels > 0 {
		args = injectBeforeOutput(args, "-ac", strconv.Itoa(opts.Channels))
	}
	if opts.BitDepth >= 16 && isLosslessOutputFormat(opts.Format) {
		args = injectBeforeOutput(args, "-sample_fmt", bitDepthToSampleFmt(opts.BitDepth))
	}
	return args
}

// injectBeforeOutput inserts a flag and value before the trailing "-" (stdout output).
func injectBeforeOutput(args []string, flag, value string) []string {
	if len(args) > 0 && args[len(args)-1] == "-" {
		result := make([]string, 0, len(args)+2)
		result = append(result, args[:len(args)-1]...)
		result = append(result, flag, value, "-")
		return result
	}
	return append(args, flag, value)
}

// isLosslessOutputFormat returns true if the format is a lossless audio format
// where preserving bit depth via -sample_fmt is meaningful.
func isLosslessOutputFormat(format string) bool {
	switch strings.ToLower(format) {
	case "flac", "alac", "wav", "aiff":
		return true
	}
	return false
}

// bitDepthToSampleFmt converts a bit depth value to the ffmpeg sample_fmt string.
// FLAC only supports s16 and s32; for 24-bit sources, s32 is the correct format
// (ffmpeg packs 24-bit samples into 32-bit containers).
func bitDepthToSampleFmt(bitDepth int) string {
	switch bitDepth {
	case 16:
		return "s16"
	case 32:
		return "s32"
	default:
		// 24-bit and other depths: use s32 (the next valid container size)
		return "s32"
	}
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
