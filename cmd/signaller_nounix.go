//go:build windows || plan9

package cmd

import (
	"context"
	"os"
	"syscall"
)

// SIGHUP is kept as a shutdown signal here, as on Windows it is delivered when the console
// window is closed, and there is no log rotation convention based on it.
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGABRT}

// Windows and Plan9 don't support SIGUSR1, so we don't need to start a signaler
func startSignaller(ctx context.Context) func() error {
	return func() error {
		return nil
	}
}
