// +build !windows
// +build !plan9

package cmd

import (
	"os"
	"os/signal"
	"syscall"
)

func init() {
	signals := []os.Signal{
		syscall.SIGUSR1,
	}
	signal.Notify(sigChan, signals...)
}
