//go:build !linux && !darwin && !windows

package local

import "github.com/rjeczalik/notify"

const WatchEvents = notify.All
