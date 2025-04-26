//go:build wasip1

package main

import "github.com/navidrome/navidrome/plugins/api"

// Required by Go WASI build
func main() {}

// Register the MediaMetadataService implementation
func init() {
	api.RegisterMediaMetadataService(MultiPlugin{})
	api.RegisterTimerCallbackService(MultiPlugin{})
}
