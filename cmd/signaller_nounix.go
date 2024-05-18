//go:build windows || plan9

package cmd

import (
	"context"
)

// Windows and Plan9 don't support SIGUSR1, so we don't need to start a signaler
func startSignaller(ctx context.Context) func() error {
	return func() error {
		return nil
	}
}
