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

const (
	// argLengthThreshold is the threshold for switching from command-line args to file-based target passing.
	// Set conservatively at 24KB to support Windows (~32KB limit) with margin for env vars.
	argLengthThreshold = 24 * 1024
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
		targetArgs, cleanup, err := targetArguments(ctx, targets, argLengthThreshold)
		if err != nil {
			progress <- &ProgressInfo{Error: err.Error()}
			return
		}
		defer cleanup()
		log.Debug(ctx, "Spawning external scanner process with target file", "fullScan", fullScan, "path", exe, "numTargets", len(targets))
		args = append(args, targetArgs...)
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

// targetArguments builds command-line arguments for the given scan targets.
// If the estimated argument length exceeds a threshold, it writes the targets to a temp file
// and returns the --target-file argument instead.
// Returns the arguments, a cleanup function to remove any temp file created, and an error if any.
func targetArguments(ctx context.Context, targets []model.ScanTarget, lengthThreshold int) ([]string, func(), error) {
	var args []string

	// Estimate argument length to decide whether to use file-based approach
	argLength := estimateArgLength(targets)

	if argLength > lengthThreshold {
		// Write targets to temp file and pass via --target-file
		targetFile, err := writeTargetsToFile(targets)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to write targets to file: %w", err)
		}
		args = append(args, "--target-file", targetFile)
		return args, func() {
			os.Remove(targetFile) // Clean up temp file
		}, nil
	}

	// Use command-line arguments for small target lists
	for _, target := range targets {
		args = append(args, "-t", target.String())
	}
	return args, func() {}, nil
}

// estimateArgLength estimates the total length of command-line arguments for the given targets.
func estimateArgLength(targets []model.ScanTarget) int {
	length := 0
	for _, target := range targets {
		// Each target adds: "-t " + target string + space
		length += 3 + len(target.String()) + 1
	}
	return length
}

// writeTargetsToFile writes the targets to a temporary file, one per line.
// Returns the path to the temp file, which the caller should clean up.
func writeTargetsToFile(targets []model.ScanTarget) (string, error) {
	tmpFile, err := os.CreateTemp("", "navidrome-scan-targets-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	for _, target := range targets {
		if _, err := fmt.Fprintln(tmpFile, target.String()); err != nil {
			os.Remove(tmpFile.Name())
			return "", fmt.Errorf("failed to write to temp file: %w", err)
		}
	}

	return tmpFile.Name(), nil
}

var _ scanner = (*scannerExternal)(nil)
