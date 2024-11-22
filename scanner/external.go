package scanner

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/navidrome/navidrome/log"
	. "github.com/navidrome/navidrome/utils/gg"
)

type scannerExternal struct {
	rootCtx context.Context
}

func (s *scannerExternal) scanAll(requestCtx context.Context, fullRescan bool, progress chan<- *scannerStatus) {
	ex, err := os.Executable()
	if err != nil {
		progress <- &scannerStatus{err: fmt.Errorf("failed to get executable path: %w", err)}
		return
	}
	log.Debug(requestCtx, "Spawning external scanner process", "fullRescan", fullRescan, "path", ex)
	cmd := exec.CommandContext(s.rootCtx, ex, "scan", "--nobanner", "--noconfig", If(fullRescan, "--full", ""))

	//in, out := io.Pipe()
	cmd.Stderr = os.Stderr
	//cmd.Stdout = out

	if err := cmd.Start(); err != nil {
		progress <- &scannerStatus{err: fmt.Errorf("failed to start scanner process: %w", err)}
		return
	}

	//go func() {
	//	sc := bufio.NewScanner(in)
	//	for sc.Scan() {
	//		fmt.Println("!!!!", sc.Text())
	//	}
	//}()

	if err := cmd.Wait(); err != nil {
		progress <- &scannerStatus{err: fmt.Errorf("scanner process failed: %w", err)}
		return
	}
}

var _ scanner = (*scannerExternal)(nil)
