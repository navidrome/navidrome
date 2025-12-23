// Test plugin for SubsonicAPI host function integration tests.
// Build with: tinygo build -o ../fake-subsonicapi-plugin.wasm -target wasip1 -buildmode=c-shared ./main.go
package main

import (
	"encoding/json"

	"github.com/extism/go-pdk"
)

type Manifest struct {
	Name        string       `json:"name"`
	Author      string       `json:"author"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

type Permissions struct {
	SubsonicAPI *SubsonicAPIPermission `json:"subsonicapi,omitempty"`
}

type SubsonicAPIPermission struct {
	Reason           string   `json:"reason,omitempty"`
	AllowedUsernames []string `json:"allowedUsernames,omitempty"`
	AllowAdmins      bool     `json:"allowAdmins,omitempty"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:        "Fake SubsonicAPI Plugin",
		Author:      "Navidrome Test",
		Version:     "1.0.0",
		Description: "Test plugin for SubsonicAPI host function",
		Permissions: &Permissions{
			SubsonicAPI: &SubsonicAPIPermission{
				Reason:           "Testing SubsonicAPI access",
				AllowedUsernames: nil, // Allow all users
				AllowAdmins:      true,
			},
		},
	}
	output, err := json.Marshal(manifest)
	if err != nil {
		pdk.SetErrorString("failed to marshal manifest")
		return 1
	}
	pdk.Output(output)
	return 0
}

// call_subsonic_api is the exported function that tests the SubsonicAPI host function.
// Input: URI string (e.g., "/ping?u=testuser")
// Output: The raw JSON response from the Subsonic API
//
//go:wasmexport call_subsonic_api
func callSubsonicAPIExport() int32 {
	// Get the URI from input
	uri := pdk.InputString()

	// Call the Subsonic API via host function
	response, err := SubsonicAPICall(uri)
	if err != nil {
		pdk.SetErrorString("failed to call SubsonicAPI: " + err.Error())
		return 1
	}

	// Return the response
	pdk.OutputString(response.ResponseJSON)
	return 0
}

func main() {}
