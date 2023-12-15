//go:build !go1.21

package scanner

import (
	"context"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
)

// TODO Remove this file when we drop support for go 1.20
func (s *scanner) RescanAll(ctx context.Context, fullRescan bool) error {
	ctx = context.TODO()
	if !isScanning.TryLock() {
		log.Debug("Scanner already running, ignoring request for rescan.")
		return ErrAlreadyScanning
	}
	defer isScanning.Unlock()

	var hasError bool
	for folder := range s.folders {
		err := s.rescan(ctx, folder, fullRescan)
		hasError = hasError || err != nil
	}
	if hasError {
		log.Error("Errors while scanning media. Please check the logs")
		core.WriteAfterScanMetrics(ctx, s.ds, false)
		return ErrScanError
	}
	core.WriteAfterScanMetrics(ctx, s.ds, true)
	return nil
}
