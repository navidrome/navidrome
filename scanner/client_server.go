package scanner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/navidrome/navidrome/log"
	. "github.com/navidrome/navidrome/utils/gg"
)

type scannerClient struct {
	rootCtx context.Context
	running sync.Mutex
}

func (s *scannerClient) ScanAll(requestCtx context.Context, fullRescan bool) error {
	if !s.running.TryLock() {
		log.Debug(requestCtx, "Scanner already running, ignoring request for rescan.")
		return ErrAlreadyScanning
	}
	defer s.running.Unlock()

	ex, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	log.Debug(requestCtx, "Spawning external scanner process", "fullRescan", fullRescan, "path", ex)
	cmd := exec.CommandContext(s.rootCtx, ex, "scan", "--nobanner", "--noconfig", If(fullRescan, "--full", ""))

	//in, out := io.Pipe()
	cmd.Stderr = os.Stderr
	//cmd.Stdout = out

	if err := cmd.Start(); err != nil {
		return err
	}

	//go func() {
	//	sc := bufio.NewScanner(in)
	//	for sc.Scan() {
	//		fmt.Println("!!!!", sc.Text())
	//	}
	//}()

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func (s *scannerClient) Status(context.Context) (*StatusInfo, error) {
	return &StatusInfo{}, nil
}

var _ Scanner = (*scannerClient)(nil)
