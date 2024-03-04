package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// NOTE: ffmpeg has a bug (https://trac.ffmpeg.org/ticket/8569) with write streaming cut from flac to flack.
//
//	It set incorrect duration and start timestamp (take it from source file)
//
// We use workaround — write output to temp file with re-encoding, not copying stream, and then stream this file.
// It's slow down transcoding performance.
// When stream into other format (opus for example no issue found)
const intermediateSubTrackTranscodingFormat = "flac"

type FFmpeg interface {
	Transcode(ctx context.Context, command string, format string, mf *model.MediaFile, maxBitRate int, beginOffset int) (io.ReadCloser, error)
	ExtractImage(ctx context.Context, path string) (io.ReadCloser, error)
	ConvertToWAV(ctx context.Context, path string) (io.ReadCloser, error)
	ConvertToFLAC(ctx context.Context, path string) (io.ReadCloser, error)
	Probe(ctx context.Context, files []string) (string, error)
	CmdPath() (string, error)
	IsAvailable() bool
	Version() string
}

func New() FFmpeg {
	return &ffmpeg{}
}

const (
	extractImageCmd        = "ffmpeg -i %s -an -vcodec copy -f image2pipe -"
	probeCmd               = "ffmpeg %s -f ffmetadata"
	createWavCmd           = "ffmpeg -i %s -c:a pcm_s16le -f wav -"
	createFLACCmd          = "ffmpeg -i %s -f flac -"
	DefaultRawTranscodeCmd = "ffmpeg -v 0 -i %s -map 0:a:0 -vn -"
)

type ffmpeg struct{}

func disableStreamCopy(mf *model.MediaFile, format string) bool {
	return mf.Suffix == format && format == intermediateSubTrackTranscodingFormat && isSubTrack(mf)
}

func forceStreamCopy(mf *model.MediaFile, format string) bool {
	if isSubTrack(mf) {
		return mf.Suffix == format && format != intermediateSubTrackTranscodingFormat
	}
	return false
}

func (e *ffmpeg) Transcode(ctx context.Context, command string, format string, mf *model.MediaFile, maxBitRate int, beginOffset int) (io.ReadCloser, error) {
	if _, err := ffmpegCmd(); err != nil {
		return nil, err
	}

	sourcePath := mf.Path
	intermediatePath := ""
	if disableStreamCopy(mf, format) {
		var err error
		intermediatePath, err = os.MkdirTemp("", "intermediate-")
		if err != nil {
			return nil, err
		}
		intermediatePath = path.Join(intermediatePath, fmt.Sprintf("media-%s-%d", mf.ID, mf.SubTrack))
	}

	log.Trace("open WV")
	fileReader := openWVFromISOMedia(mf)
	if fileReader != nil {
		log.Trace("use WV")
		sourcePath = "-"
	}
	args := createFFmpegCommandForMedia(command, format, sourcePath, intermediatePath, mf, maxBitRate, beginOffset)

	return e.start(ctx, args, fileReader, intermediatePath)
}

func (e *ffmpeg) ExtractImage(ctx context.Context, path string) (io.ReadCloser, error) {
	if _, err := ffmpegCmd(); err != nil {
		return nil, err
	}
	args := createFFmpegCommand(extractImageCmd, path)
	fileReader := openWVFromISO(path)
	if fileReader != nil {
		for i, s := range args {
			if s == path {
				args[i] = "-"
			}
		}
	}
	return e.start(ctx, args, fileReader, "")
}

func (e *ffmpeg) ConvertToWAV(ctx context.Context, path string) (io.ReadCloser, error) {
	args := createFFmpegCommand(createWavCmd, path)
	return e.start(ctx, args, nil, "")
}

func (e *ffmpeg) ConvertToFLAC(ctx context.Context, path string) (io.ReadCloser, error) {
	args := createFFmpegCommand(createFLACCmd, path)
	return e.start(ctx, args, nil, "")
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

func (e *ffmpeg) start(ctx context.Context, args []string, fileReader io.ReadCloser, tempFilePath string) (io.ReadCloser, error) {
	log.Trace(ctx, "Executing ffmpeg command", "cmd", args)
	j := &ffCmd{args: args, intermediatePath: tempFilePath}
	j.PipeReader, j.out = io.Pipe()
	j.in = fileReader
	err := j.start()
	if err != nil {
		return nil, err
	}
	go j.wait()
	return j, nil
}

type ffCmd struct {
	*io.PipeReader
	out              *io.PipeWriter
	args             []string
	cmd              *exec.Cmd
	intermediatePath string
	in               io.ReadCloser
	inWriter         io.WriteCloser
}

func (j *ffCmd) start() error {
	cmd := exec.Command(j.args[0], j.args[1:]...) // #nosec
	if j.intermediatePath == "" {
		cmd.Stdout = j.out
	}

	if log.IsGreaterOrEqualTo(log.LevelTrace) {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = io.Discard
	}
	j.cmd = cmd

	if j.in != nil {
		var err error
		j.inWriter, err = cmd.StdinPipe()
		if err != nil {
			_ = j.in.Close()
			log.Error("Can't open ffmpeg stdin pipe", "error", err)
			return err
		}
	}

	if err := j.cmd.Start(); err != nil {
		return fmt.Errorf("starting cmd: %w", err)
	}
	return nil
}

func (j *ffCmd) streamToInput() {
	defer func() {
		log.Debug("close pipe source")
		_ = j.in.Close()
		_ = j.inWriter.Close()
	}()

	wrote, err := io.Copy(j.inWriter, j.in)
	if err != nil {
		log.Error("Can't read from iso", "error", err)
		return
	}
	log.Debug(fmt.Sprintf("Wrote data: %d", wrote))
}

func (j *ffCmd) wait() {
	if j.intermediatePath != "" {
		defer func() {
			_ = os.RemoveAll(path.Dir(j.intermediatePath))
		}()
	}
	if j.inWriter != nil {
		j.streamToInput()
	}
	if err := j.cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_ = j.out.CloseWithError(fmt.Errorf("%s exited with non-zero status code: %d", j.args[0], exitErr.ExitCode()))
		} else {
			_ = j.out.CloseWithError(fmt.Errorf("waiting %s cmd: %w", j.args[0], err))
		}
		return
	}
	if j.intermediatePath != "" {
		f, err := os.Open(j.intermediatePath)
		if err != nil {
			_ = j.out.CloseWithError(fmt.Errorf("failed to open intermediate media '%s': %w", j.intermediatePath, err))
			return
		}
		_, err = io.Copy(j.out, f)
		if err != nil {
			_ = j.out.CloseWithError(fmt.Errorf("failed to copy data from intermediate media '%s': %w", j.intermediatePath, err))
			return
		}
		_ = f.Close()
	}
	_ = j.out.Close()
}

func isSubTrack(mf *model.MediaFile) bool {
	return mf.SubTrack > -1
}

func makeMetadataParams(mf *model.MediaFile) []string {
	var result []string

	// Fill metadata only for multi-track media
	if !isSubTrack(mf) {
		return nil
	}

	result = append(result, "-metadata", fmt.Sprintf(`title=%s`, mf.Title))
	result = append(result, "-metadata", fmt.Sprintf(`artist=%s`, mf.Artist))
	result = append(result, "-metadata", fmt.Sprintf(`album=%s`, mf.Album))
	if mf.AlbumArtist != "" {
		result = append(result, "-metadata", fmt.Sprintf(`album_artist=%s`, mf.AlbumArtist))
	}
	result = append(result, "-metadata", fmt.Sprintf("year=%d", mf.Year))
	result = append(result, "-metadata", fmt.Sprintf("track=%d", mf.TrackNumber))
	if mf.DiscNumber > 0 {
		result = append(result, "-metadata", fmt.Sprintf("disc=%d", mf.DiscNumber))
	}
	result = append(result, "-metadata", fmt.Sprintf(`comment=%s`, mf.Comment))
	result = append(result, "-metadata", fmt.Sprintf(`genre=%s`, mf.Genre))

	if mf.RgAlbumGain != 0.0 {
		result = append(result, "-metadata", fmt.Sprintf("replaygain_album_gain=%f", mf.RgAlbumGain))
	}
	if mf.RgAlbumPeak != 0.0 {
		result = append(result, "-metadata", fmt.Sprintf("replaygain_album_peak=%f", mf.RgAlbumPeak))
	}
	if mf.RgTrackGain != 0.0 {
		result = append(result, "-metadata", fmt.Sprintf("replaygain_track_gain=%f", mf.RgTrackGain))
	}
	if mf.RgTrackPeak != 0.0 {
		result = append(result, "-metadata", fmt.Sprintf("replaygain_track_peak=%f", mf.RgTrackPeak))
	}

	result = append(result, "-metadata", "cuesheet=")

	return result
}

var zeroTime = time.Unix(0, 0).UTC()

func makeTime(timeInSeconds float32) string {
	return zeroTime.Add(time.Duration(timeInSeconds*1000.0) * time.Millisecond).Format("15:04:05.000")
}

func makeRangeParams(mf *model.MediaFile, extraOffset int) []string {
	var result []string

	// Insert range only for multi-track media
	if isSubTrack(mf) {
		if mf.Offset+float32(extraOffset) > 0.0 {
			result = append(result, "-ss", makeTime(mf.Offset+float32(extraOffset)))
		}
		if mf.Duration > 0.0 {
			result = append(result, "-t", makeTime(mf.Duration))
		}
	}

	return result
}

// Path will always be an absolute path
func createFFmpegCommand(cmd, path string) []string {
	split := strings.Split(cmd, " ")
	for i, s := range split {
		switch s {
		case "%s":
			split[i] = path
		default:
		}
	}

	return split
}

func createFFmpegCommandForMedia(cmd string, format string, sourcePath string, intermediatePath string, mf *model.MediaFile, maxBitRate int, beginOffset int) []string {
	offsetIndex := -1
	destFormat := ""
	copyIndex := -1

	var args []string

	split := strings.Split(cmd, " ")
	for i, s := range split {
		var preArgs []string
		var postArgs []string
		switch s {
		case "-vn":
			copyIndex = len(args)
		case "-i":
			// Need insert start offset and duration before input file
			if isSubTrack(mf) {
				preArgs = makeRangeParams(mf, beginOffset)
			}
		case "-":
			if destFormat == "" {
				destFormat = format
				preArgs = append(preArgs, "-f", destFormat)
			}
			// Insert metadata before last param for sub-tracks
			if isSubTrack(mf) {
				preArgs = append(preArgs, makeMetadataParams(mf)...)
				if intermediatePath != "" {
					s = intermediatePath
				}
			}
		case "%t":
			if !isSubTrack(mf) {
				s = fmt.Sprintf("%v", beginOffset)
				offsetIndex = -1
			}
		case "%s":
			s = sourcePath // mf.Path or '-'
			if beginOffset > 0 && !isSubTrack(mf) {
				offsetIndex = len(args) + 1
			}
		case "-f":
			destFormat = split[i+1]
		default:
			if strings.Contains(s, "%b") {
				s = strings.ReplaceAll(s, "%b", strconv.Itoa(maxBitRate))
			}
		}

		if len(preArgs) > 0 {
			args = append(args, preArgs...)
		}
		if s != "" {
			args = append(args, s)
		}
		if len(postArgs) > 0 {
			args = append(args, postArgs...)
		}
	}

	if offsetIndex != -1 {
		offsetParams := []string{"-ss", fmt.Sprintf("%v", beginOffset)}
		args = append(args[:offsetIndex], append(offsetParams, args[offsetIndex:]...)...)
	}

	if copyIndex != -1 && forceStreamCopy(mf, destFormat) {
		args = append(args[:copyIndex], append([]string{"-c:a copy"}, args[copyIndex:]...)...)
	}

	return args
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
