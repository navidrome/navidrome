package scanner

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// scannerExternal is a scanner that runs an external process to do the scanning. It is used to avoid
// memory leaks or retention in the main process, as the scanner can consume a lot of memory. The
// external process will be spawned with the same executable as the current process, and will run
// the "scan" command with the "--subprocess" flag.
//
// The external process will send progress updates to the main process through its STDOUT, and the main
// process will forward them to the caller.
type scannerExternal struct{}

func (s *scannerExternal) scanFolders(ctx context.Context, fullScan bool, targets []model.ScanTarget, progress chan<- *ProgressInfo) {
	s.scan(ctx, fullScan, targets, progress)
}

func (s *scannerExternal) scan(ctx context.Context, fullScan bool, targets []model.ScanTarget, progress chan<- *ProgressInfo) {
	exe, err := os.Executable()
	if err != nil {
		progress <- &ProgressInfo{Error: fmt.Sprintf("failed to get executable path: %s", err)}
		return
	}

	// Build command arguments
	args := []string{
		"scan",
		"--nobanner", "--subprocess",
		"--configfile", conf.Server.ConfigFile,
		"--datafolder", conf.Server.DataFolder,
		"--cachefolder", conf.Server.CacheFolder,
	}

	// Add targets if provided
	if len(targets) > 0 {
		for _, target := range targets {
			args = append(args, "-t", target.String())
		}
		log.Debug(ctx, "Spawning external scanner process with targets", "fullScan", fullScan, "path", exe, "targets", targets)
	} else {
		log.Debug(ctx, "Spawning external scanner process", "fullScan", fullScan, "path", exe)
	}

	// Add full scan flag if needed
	if fullScan {
		args = append(args, "--full")
	}

	cmd := exec.CommandContext(ctx, exe, args...)

	in, out := io.Pipe()
	defer in.Close()
	defer out.Close()
	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		progress <- &ProgressInfo{Error: fmt.Sprintf("failed to start scanner process: %s", err)}
		return
	}
	go s.wait(cmd, out)

	decoder := gob.NewDecoder(in)
	for {
		var p ProgressInfo
		if err := decoder.Decode(&p); err != nil {
			if !errors.Is(err, io.EOF) {
				progress <- &ProgressInfo{Error: fmt.Sprintf("failed to read status from scanner: %s", err)}
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
