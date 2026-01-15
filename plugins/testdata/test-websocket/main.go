// Test WebSocket plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-websocket.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"errors"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/websocket"
)

func init() {
	websocket.Register(&testWebSocket{})
}

type testWebSocket struct{}

// OnTextMessage is called when a text message is received.
// Magic messages trigger specific behaviors to test host functions:
//   - "echo": sends back the same message using SendText host function
//   - "close": closes the connection using CloseConnection host function
//   - "store:MESSAGE": stores MESSAGE in plugin config for later retrieval
//   - "fail": returns an error to test error handling
func (t *testWebSocket) OnTextMessage(input websocket.OnTextMessageRequest) error {
	// Store all received messages for test verification
	storeReceivedMessage("text:" + input.Message)

	switch input.Message {
	case "echo":
		if err := host.WebSocketSendText(input.ConnectionID, "echo:"+input.Message); err != nil {
			return err
		}

	case "close":
		if err := host.WebSocketCloseConnection(input.ConnectionID, 1000, "closed by plugin"); err != nil {
			return err
		}

	case "fail":
		return errors.New("intentional test failure")
	}

	return nil
}

// OnBinaryMessage is called when a binary message is received.
func (t *testWebSocket) OnBinaryMessage(input websocket.OnBinaryMessageRequest) error {
	// Store received binary data for test verification
	storeReceivedMessage("binary:" + input.Data)
	return nil
}

// OnError is called when an error occurs on a WebSocket connection.
func (t *testWebSocket) OnError(input websocket.OnErrorRequest) error {
	// Store error for test verification
	storeReceivedMessage("error:" + input.Error)
	return nil
}

// OnClose is called when a WebSocket connection is closed.
func (t *testWebSocket) OnClose(input websocket.OnCloseRequest) error {
	// Store close event for test verification
	storeReceivedMessage("close:" + input.Reason)
	return nil
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
