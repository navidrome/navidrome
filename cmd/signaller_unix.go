//go:build !windows && !plan9

package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const triggerScanSignal = syscall.SIGUSR1

// shutdownSignals does not include SIGHUP: as expected from a daemon, the server handles it
// by reopening its log file (see handleSignal), so log rotation tools can use it.
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGABRT}

func startSignaller(ctx context.Context) func() error {
	log.Info(ctx, "Starting signaler")
	scanner := CreateScanner(ctx)

	return func() error {
		var sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan, triggerScanSignal, syscall.SIGHUP)
		defer signal.Stop(sigChan)

		for {
			select {
			case sig := <-sigChan:
				handleSignal(ctx, sig, scanner)
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func handleSignal(ctx context.Context, sig os.Signal, scanner model.Scanner) {
	switch sig {
	case syscall.SIGHUP:
		reopenLogFile(ctx, sig)
	case triggerScanSignal:
		log.Info(ctx, "Received signal, triggering a new scan", "signal", sig)
		start := time.Now()
		_, err := scanner.ScanAll(ctx, false)
		if err != nil {
			log.Error(ctx, "Error scanning", err)
		}
		log.Info(ctx, "Triggered scan complete", "elapsed", time.Since(start))
	}
}

func reopenLogFile(ctx context.Context, sig os.Signal) {
	if conf.Server.LogFile == "" {
		log.Debug(ctx, "Received signal, but no log file configured. Ignoring", "signal", sig)
		return
	}
	log.Info(ctx, "Received signal, reopening log file", "signal", sig, "logFile", conf.Server.LogFile)
	if err := log.SetOutputFile(conf.Server.LogFile); err != nil {
		log.Error(ctx, "Error reopening log file", "logFile", conf.Server.LogFile, err)
	}
}
