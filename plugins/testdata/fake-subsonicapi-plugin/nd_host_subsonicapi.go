package main

import (
	"encoding/json"
	"errors"

	"github.com/extism/go-pdk"
)

// subsonicapiCall is the host function provided by Navidrome to call the Subsonic API.
// It takes a URI string and returns a JSON response.
//
//go:wasmimport extism:host/user subsonicapi_call
func subsonicapiCall(uri uint64) uint64

// SubsonicAPICallResponse matches the host response format.
type SubsonicAPICallResponse struct {
	ResponseJSON string `json:"responseJSON,omitempty"`
	Error        string `json:"error,omitempty"`
}

// SubsonicAPICall is a wrapper around the host subsonicapi_call function.
func SubsonicAPICall(uri string) (*SubsonicAPICallResponse, error) {
	// Allocate memory for the URI string
	mem := pdk.AllocateString(uri)
	defer mem.Free()

	// Call the host function
	responsePtr := subsonicapiCall(mem.Offset())

	// Read the response from memory
	responseMem := pdk.FindMemory(responsePtr)
	responseBytes := responseMem.ReadBytes()

	// Parse the response
	var response SubsonicAPICallResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, err
	}

	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	return &response, nil
}
