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

func startSignaler(ctx context.Context) func() error {
	log.Info(ctx, "Starting signaler")
	scanner := GetScanner()

	return func() error {
		var sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan, triggerScanSignal)

		for {
			select {
			case sig := <-sigChan:
				log.Info(ctx, "Received signal, triggering a new scan", "signal", sig)
				start := time.Now()
				err := scanner.RescanAll(ctx, false)
				if err != nil {
					log.Error(ctx, "Error scanning", err)
				}
				log.Info(ctx, "Triggered scan complete", "elapsed", time.Since(start).Round(100*time.Millisecond))
			case <-ctx.Done():
				return nil
			}
		}
	}
}
