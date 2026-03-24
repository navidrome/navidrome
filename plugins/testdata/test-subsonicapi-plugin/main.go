// Test plugin for SubsonicAPI host function integration tests.
// Build with: tinygo build -o ../test-subsonicapi-plugin.wasm -target wasip1 -buildmode=c-shared ./main.go
package main

import (
	"fmt"

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

// call_subsonic_api_raw is the exported function that tests the SubsonicAPI CallRaw host function.
// Input: URI string (e.g., "/getCoverArt?u=testuser&id=al-1")
// Output: JSON with contentType, size, and first bytes of the raw response
//
//go:wasmexport call_subsonic_api_raw
func callSubsonicAPIRawExport() int32 {
	uri := pdk.InputString()

	contentType, data, err := host.SubsonicAPICallRaw(uri)
	if err != nil {
		pdk.SetErrorString("failed to call SubsonicAPI raw: " + err.Error())
		return 1
	}

	// Return metadata about the raw response as JSON
	firstByte := 0
	if len(data) > 0 {
		firstByte = int(data[0])
	}
	result := fmt.Sprintf(`{"contentType":%q,"size":%d,"firstByte":%d}`, contentType, len(data), firstByte)
	pdk.OutputString(result)
	return 0
}

func main() {}
