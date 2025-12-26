// Fake WebSocket plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../fake-websocket.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"encoding/json"

	pdk "github.com/extism/go-pdk"
)

// Manifest types
type Manifest struct {
	Name        string       `json:"name"`
	Author      string       `json:"author"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

type Permissions struct {
	WebSocket *WebSocketPermission `json:"websocket,omitempty"`
}

type WebSocketPermission struct {
	Reason       string   `json:"reason,omitempty"`
	AllowedHosts []string `json:"allowedHosts,omitempty"`
}

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:        "Fake WebSocket",
		Author:      "Navidrome Test",
		Version:     "1.0.0",
		Description: "A fake WebSocket plugin for integration testing",
		Permissions: &Permissions{
			WebSocket: &WebSocketPermission{
				Reason:       "For testing WebSocket callbacks",
				AllowedHosts: []string{"*.example.com", "localhost:*", "echo.websocket.org"},
			},
		},
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	pdk.Output(out)
	return 0
}

// OnTextMessageInput is the input for nd_websocket_on_text_message callback.
type OnTextMessageInput struct {
	ConnectionID string `json:"connection_id"`
	Message      string `json:"message"`
}

// OnTextMessageOutput is the output from nd_websocket_on_text_message callback.
type OnTextMessageOutput struct {
	Error *string `json:"error,omitempty"`
}

// nd_websocket_on_text_message is called when a text message is received.
// Magic messages trigger specific behaviors to test host functions:
//   - "echo": sends back the same message using SendText host function
//   - "close": closes the connection using CloseConnection host function
//   - "store:MESSAGE": stores MESSAGE in plugin config for later retrieval
//   - "fail": returns an error to test error handling
//
//go:wasmexport nd_websocket_on_text_message
func ndWebSocketOnTextMessage() int32 {
	var input OnTextMessageInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(OnTextMessageOutput{Error: &errStr})
		return 0
	}

	// Store all received messages for test verification
	storeReceivedMessage("text:" + input.Message)

	switch input.Message {
	case "echo":
		_, err := WebSocketSendText(input.ConnectionID, "echo:"+input.Message)
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(OnTextMessageOutput{Error: &errStr})
			return 0
		}

	case "close":
		_, err := WebSocketCloseConnection(input.ConnectionID, 1000, "closed by plugin")
		if err != nil {
			errStr := err.Error()
			pdk.OutputJSON(OnTextMessageOutput{Error: &errStr})
			return 0
		}

	case "fail":
		errStr := "intentional test failure"
		pdk.OutputJSON(OnTextMessageOutput{Error: &errStr})
		return 0
	}

	pdk.OutputJSON(OnTextMessageOutput{})
	return 0
}

// OnBinaryMessageInput is the input for nd_websocket_on_binary_message callback.
type OnBinaryMessageInput struct {
	ConnectionID string `json:"connection_id"`
	Data         string `json:"data"` // Base64 encoded
}

// OnBinaryMessageOutput is the output from nd_websocket_on_binary_message callback.
type OnBinaryMessageOutput struct {
	Error *string `json:"error,omitempty"`
}

// nd_websocket_on_binary_message is called when a binary message is received.
//
//go:wasmexport nd_websocket_on_binary_message
func ndWebSocketOnBinaryMessage() int32 {
	var input OnBinaryMessageInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(OnBinaryMessageOutput{Error: &errStr})
		return 0
	}

	// Store received binary data for test verification
	storeReceivedMessage("binary:" + input.Data)

	pdk.OutputJSON(OnBinaryMessageOutput{})
	return 0
}

// OnErrorInput is the input for nd_websocket_on_error callback.
type OnErrorInput struct {
	ConnectionID string `json:"connection_id"`
	Error        string `json:"error"`
}

// OnErrorOutput is the output from nd_websocket_on_error callback.
type OnErrorOutput struct {
	Error *string `json:"error,omitempty"`
}

// nd_websocket_on_error is called when an error occurs on a WebSocket connection.
//
//go:wasmexport nd_websocket_on_error
func ndWebSocketOnError() int32 {
	var input OnErrorInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(OnErrorOutput{Error: &errStr})
		return 0
	}

	// Store error for test verification
	storeReceivedMessage("error:" + input.Error)

	pdk.OutputJSON(OnErrorOutput{})
	return 0
}

// OnCloseInput is the input for nd_websocket_on_close callback.
type OnCloseInput struct {
	ConnectionID string `json:"connection_id"`
	Code         int    `json:"code"`
	Reason       string `json:"reason"`
}

// OnCloseOutput is the output from nd_websocket_on_close callback.
type OnCloseOutput struct {
	Error *string `json:"error,omitempty"`
}

// nd_websocket_on_close is called when a WebSocket connection is closed.
//
//go:wasmexport nd_websocket_on_close
func ndWebSocketOnClose() int32 {
	var input OnCloseInput
	if err := pdk.InputJSON(&input); err != nil {
		errStr := err.Error()
		pdk.OutputJSON(OnCloseOutput{Error: &errStr})
		return 0
	}

	// Store close event for test verification
	storeReceivedMessage("close:" + input.Reason)

	pdk.OutputJSON(OnCloseOutput{})
	return 0
}

// storeReceivedMessage stores messages in plugin variable storage for test verification.
// Messages are appended to an existing list.
func storeReceivedMessage(msg string) {
	// Use Extism var storage for plugin state
	if existingVar := pdk.GetVar("_received_messages"); existingVar != nil {
		msg = string(existingVar) + "\n" + msg
	}
	pdk.SetVar("_received_messages", []byte(msg))
}

func main() {}
