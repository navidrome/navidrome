//go:build !linux

package local

import "github.com/rjeczalik/notify"

// BFR: Need to support all other platforms
const WatchEvents = notify.All
