// Test WebSocket plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-websocket.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"errors"

	pdk "github.com/extism/go-pdk"
)

// OnTextMessageInput is the input for nd_websocket_on_text_message callback.
type OnTextMessageInput struct {
	ConnectionID string `json:"connectionId"`
	Message      string `json:"message"`
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
		pdk.SetError(err)
		return -1
	}

	// Store all received messages for test verification
	storeReceivedMessage("text:" + input.Message)

	switch input.Message {
	case "echo":
		if _, err := WebSocketSendText(input.ConnectionID, "echo:"+input.Message); err != nil {
			pdk.SetError(err)
			return -1
		}

	case "close":
		if _, err := WebSocketCloseConnection(input.ConnectionID, 1000, "closed by plugin"); err != nil {
			pdk.SetError(err)
			return -1
		}

	case "fail":
		pdk.SetError(errors.New("intentional test failure"))
		return -1
	}

	return 0
}

// OnBinaryMessageInput is the input for nd_websocket_on_binary_message callback.
type OnBinaryMessageInput struct {
	ConnectionID string `json:"connectionId"`
	Data         string `json:"data"` // Base64 encoded
}

// nd_websocket_on_binary_message is called when a binary message is received.
//
//go:wasmexport nd_websocket_on_binary_message
func ndWebSocketOnBinaryMessage() int32 {
	var input OnBinaryMessageInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return -1
	}

	// Store received binary data for test verification
	storeReceivedMessage("binary:" + input.Data)

	return 0
}

// OnErrorInput is the input for nd_websocket_on_error callback.
type OnErrorInput struct {
	ConnectionID string `json:"connectionId"`
	Error        string `json:"error"`
}

// nd_websocket_on_error is called when an error occurs on a WebSocket connection.
//
//go:wasmexport nd_websocket_on_error
func ndWebSocketOnError() int32 {
	var input OnErrorInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return -1
	}

	// Store error for test verification
	storeReceivedMessage("error:" + input.Error)

	return 0
}

// OnCloseInput is the input for nd_websocket_on_close callback.
type OnCloseInput struct {
	ConnectionID string `json:"connectionId"`
	Code         int    `json:"code"`
	Reason       string `json:"reason"`
}

// nd_websocket_on_close is called when a WebSocket connection is closed.
//
//go:wasmexport nd_websocket_on_close
func ndWebSocketOnClose() int32 {
	var input OnCloseInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return -1
	}

	// Store close event for test verification
	storeReceivedMessage("close:" + input.Reason)

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
