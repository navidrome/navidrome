package local

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/rjeczalik/notify"
)

// Start starts a watcher on the whole FS and returns a channel to send detected changes.
// It uses `notify` to detect changes in the filesystem, so it may not work on all platforms/use-cases.
// Notoriously, it does not work on some networked mounts and Windows with WSL2.
func (s *localStorage) Start(ctx context.Context) (<-chan string, error) {
	if !s.watching.CompareAndSwap(false, true) {
		return nil, errors.New("watcher already started")
	}
	input := make(chan notify.EventInfo, 500)
	libPath := filepath.Join(s.u.Path, "...")
	log.Debug(ctx, "Starting watcher", "lib", libPath)
	if err := notify.Watch(libPath, input, WatchEvents); err != nil {
		s.watching.Store(false)
		return nil, fmt.Errorf("starting watcher on %s: %w", libPath, err)
	}

	output := make(chan string, 500)
	go func() {
		defer close(input)
		defer close(output)
		defer notify.Stop(input)

		for {
			select {
			case event := <-input:
				log.Trace(ctx, "Detected change", "event", event, "lib", s.u.Path)
				name := event.Path()
				name = strings.Replace(name, s.resolvedPath, s.u.Path, 1)
				output <- name
			case <-ctx.Done():
				log.Debug(ctx, "Stopping watcher", "path", s.u.Path)
				s.watching.Store(false)
				return
			}
		}
	}()
	return output, nil
}
