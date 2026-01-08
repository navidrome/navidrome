// Test plugin for SubsonicAPI host function integration tests.
// Build with: tinygo build -o ../test-subsonicapi-plugin.wasm -target wasip1 -buildmode=c-shared ./main.go
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

// call_subsonic_api is the exported function that tests the SubsonicAPI host function.
// Input: URI string (e.g., "/ping?u=testuser")
// Output: The raw JSON response from the Subsonic API
//
//go:wasmexport call_subsonic_api
func callSubsonicAPIExport() int32 {
	// Get the URI from input
	uri := pdk.InputString()

	// Call the Subsonic API via host function
	responseJSON, err := host.SubsonicAPICall(uri)
	if err != nil {
		pdk.SetErrorString("failed to call SubsonicAPI: " + err.Error())
		return 1
	}

	// Return the response
	pdk.OutputString(responseJSON)
	return 0
}

func main() {}
