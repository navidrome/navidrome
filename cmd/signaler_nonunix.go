//go:build windows || plan9

package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/navidrome/navidrome/log"
)

func startSignaler(ctx context.Context) func() error {
	log.Info(ctx, "Starting signaler")

	return func() error {
		var sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		select {
		case sig := <-sigChan:
			log.Info(ctx, "Received termination signal", "signal", sig)
			return interrupted
		case <-ctx.Done():
			return nil
		}
	}
}
