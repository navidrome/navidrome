package mpv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dexterlb/mpvipc"
	"github.com/kballard/go-shellquote"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

func start(ctx context.Context, args []string) (Executor, error) {
	if len(args) == 0 {
		return Executor{}, fmt.Errorf("no command arguments provided")
	}
	log.Debug("Executing mpv command", "cmd", args)
	j := Executor{args: args}
	j.PipeReader, j.out = io.Pipe()
	err := j.start(ctx)
	if err != nil {
		return Executor{}, err
	}
	go j.wait()
	return j, nil
}

func (j *Executor) Cancel() error {
	if j.cmd != nil {
		return j.cmd.Cancel()
	}
	return fmt.Errorf("there is non command to cancel")
}

type Executor struct {
	*io.PipeReader
	out  *io.PipeWriter
	args []string
	cmd  *exec.Cmd
}

func (j *Executor) start(ctx context.Context) error {
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

func (j *Executor) wait() {
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
func createMPVCommand(deviceName string, filename string, socketName string) []string {
	// Parse the template structure using shell parsing to handle quoted arguments
	templateArgs, err := shellquote.Split(conf.Server.MPVCmdTemplate)
	if err != nil {
		log.Error("Failed to parse MPV command template", "template", conf.Server.MPVCmdTemplate, err)
		return nil
	}

	// Replace placeholders in each parsed argument to preserve spaces in substituted values
	for i, arg := range templateArgs {
		arg = strings.ReplaceAll(arg, "%d", deviceName)
		arg = strings.ReplaceAll(arg, "%f", filename)
		arg = strings.ReplaceAll(arg, "%s", socketName)
		templateArgs[i] = arg
	}

	// Replace mpv executable references with the configured path
	if len(templateArgs) > 0 {
		cmdPath, err := mpvCommand()
		if err == nil {
			if templateArgs[0] == "mpv" || templateArgs[0] == "mpv.exe" {
				templateArgs[0] = cmdPath
			}
		}
	}

	return templateArgs
}

// This is a 1:1 copy of the stuff in ffmpeg.go, need to be unified.
func mpvCommand() (string, error) {
	mpvOnce.Do(func() {
		if conf.Server.MPVPath != "" {
			mpvPath = conf.Server.MPVPath
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

type MpvConnection struct {
	Conn          *mpvipc.Connection
	Exe           *Executor
	CloseCalled   bool
	IPCSocketName string
}

func (t *MpvConnection) isSocketFilePresent() bool {
	if len(t.IPCSocketName) < 1 {
		return false
	}

	fileInfo, err := os.Stat(t.IPCSocketName)
	return err == nil && fileInfo != nil && !fileInfo.IsDir()
}

func NewConnection(ctx context.Context, deviceName string) (*MpvConnection, error) {
	log.Debug("Loading mpv connection")

	if _, err := mpvCommand(); err != nil {
		return nil, err
	}

	tmpSocketName := socketName("mpv-ctrl-", ".socket")

	args := createMPVCommand(deviceName, "null", tmpSocketName)
	exe, err := start(ctx, args)
	if err != nil {
		log.Error("Error starting mpv process", err)
		return nil, err
	}

	// wait for socket to show up
	err = waitForSocket(tmpSocketName, 3*time.Second, 100*time.Millisecond)
	if err != nil {
		log.Error("Error or timeout waiting for control socket", "socketname", tmpSocketName, err)
		return nil, err
	}

	conn := mpvipc.NewConnection(tmpSocketName)
	err = conn.Open()

	if err != nil {
		log.Error("Error opening new connection", err)
		return nil, err
	}

	theConn := &MpvConnection{Conn: conn, IPCSocketName: tmpSocketName, Exe: &exe, CloseCalled: false}

	go func() {
		conn.WaitUntilClosed()
		log.Info("Hitting end-of-stream, signalling on channel")

		if !theConn.CloseCalled {
			log.Debug("Close cleanup")
			// trying to shutdown mpv process using socket
			if theConn.isSocketFilePresent() {
				log.Debug("sending shutdown command")
				_, err := theConn.Conn.Call("quit")
				if err != nil {
					log.Warn("Error sending quit command to mpv-ipc socket", err)

					if theConn.Exe != nil {
						log.Debug("cancelling executor")
						err = theConn.Exe.Cancel()
						if err != nil {
							log.Warn("Error canceling executor", err)
						}
					}
					removeSocket(theConn.IPCSocketName)
				}
			}
		}
	}()

	return theConn, nil
}

var (
	mpvOnce sync.Once
	mpvPath string
	mpvErr  error
)
