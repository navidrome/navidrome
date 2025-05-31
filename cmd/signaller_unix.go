//go:build !windows && !plan9

package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/navidrome/navidrome/log"
)

const triggerScanSignal = syscall.SIGUSR1

func startSignaller(ctx context.Context) func() error {
	log.Info(ctx, "Starting signaler")
	scanner := CreateScanner(ctx)

	return func() error {
		var sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan, triggerScanSignal)

		for {
			select {
			case sig := <-sigChan:
				log.Info(ctx, "Received signal, triggering a new scan", "signal", sig)
				start := time.Now()
				_, err := scanner.ScanAll(ctx, false)
				if err != nil {
					log.Error(ctx, "Error scanning", err)
				}
				log.Info(ctx, "Triggered scan complete", "elapsed", time.Since(start))
			case <-ctx.Done():
				return nil
			}
		}
	}
}
