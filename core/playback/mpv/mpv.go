package mpv

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

// mpv --no-audio-display --pause 'Jack Johnson/On And On/01 Times Like These.m4a' --input-ipc-server=/tmp/gonzo.socket
const (
	mpvSocket       = "/tmp/mpv_rpc"
	mpvComdTemplate = "mpv --no-audio-display --pause %s --input-unix-socket=%s"
)

func start(args []string) (io.ReadCloser, error) {
	log.Trace("Executing mpv command", "cmd", args)
	j := &mpvCmd{args: args}
	j.PipeReader, j.out = io.Pipe()
	err := j._start()
	if err != nil {
		return nil, err
	}
	go j.wait()
	return j, nil
}

type mpvCmd struct {
	*io.PipeReader
	out  *io.PipeWriter
	args []string
	cmd  *exec.Cmd
}

func (j *mpvCmd) _start() error {
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

func (j *mpvCmd) wait() {
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
func createMPVCommand(cmd, filename string, socketName string) []string {
	split := strings.Split(fixCmd(cmd), " ")
	for i, s := range split {
		s = strings.ReplaceAll(s, "%s", filename)
		s = strings.ReplaceAll(s, "%s", socketName)
		split[i] = s
	}

	return split
}

func fixCmd(cmd string) string {
	split := strings.Split(cmd, " ")
	var result []string
	cmdPath, _ := mpvCommand()
	for _, s := range split {
		if s == "mpv" || s == "mpv.exe" {
			result = append(result, cmdPath)
		} else {
			result = append(result, s)
		}
	}
	return strings.Join(result, " ")
}

// This is a 1:1 copy of the stuff in ffmpeg.go, need to be unified.
func mpvCommand() (string, error) {
	mpvOnce.Do(func() {
		if conf.Server.MPVPath != "" {
			mpvPath = conf.Server.FFmpegPath
			mpvPath, mpvErr = exec.LookPath(mpvPath)
		} else {
			mpvPath, mpvErr = exec.LookPath("mpv")
			if errors.Is(mpvErr, exec.ErrDot) {
				log.Trace("mpv found in current folder '.'")
				mpvPath, mpvErr = exec.LookPath("./mpv")
			}
		}
		if mpvErr == nil {
			log.Info("Found mpv", "path", mpvPath)
			return
		}
	})
	return mpvPath, mpvErr
}

var (
	mpvOnce sync.Once
	mpvPath string
	mpvErr  error
)
