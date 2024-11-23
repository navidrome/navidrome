package scanner

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/navidrome/navidrome/log"
	. "github.com/navidrome/navidrome/utils/gg"
)

type scannerExternal struct {
	rootCtx context.Context
}

func (s *scannerExternal) scanAll(requestCtx context.Context, fullRescan bool, progress chan<- *ProgressInfo) {
	ex, err := os.Executable()
	if err != nil {
		progress <- &ProgressInfo{Err: fmt.Errorf("failed to get executable path: %w", err)}
		return
	}
	log.Debug(requestCtx, "Spawning external scanner process", "fullRescan", fullRescan, "path", ex)
	cmd := exec.CommandContext(s.rootCtx, ex, "scan", "--nobanner", "--noconfig", If(fullRescan, "--full", ""))

	in, out := io.Pipe()
	defer in.Close()
	defer out.Close()
	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		progress <- &ProgressInfo{Err: fmt.Errorf("failed to start scanner process: %w", err)}
		return
	}
	go s.wait(cmd, out)

	decoder := gob.NewDecoder(in)
	for {
		var p ProgressInfo
		if err := decoder.Decode(&p); err != nil {
			if !errors.Is(err, io.EOF) {
				progress <- &ProgressInfo{Err: fmt.Errorf("failed to read status from scanner: %w", err)}
			}
			break
		}
		progress <- &p
	}
}

func (s *scannerExternal) wait(cmd *exec.Cmd, out *io.PipeWriter) {
	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_ = out.CloseWithError(fmt.Errorf("%s exited with non-zero status code: %w", cmd, exitErr))
		} else {
			_ = out.CloseWithError(fmt.Errorf("waiting %s cmd: %w", cmd, err))
		}
		return
	}
	_ = out.Close()
}

var _ scanner = (*scannerExternal)(nil)
