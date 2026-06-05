package host

import "context"

// WebSocketService provides WebSocket communication capabilities for plugins.
//
// This service allows plugins to establish WebSocket connections to external services,
// send and receive messages, and manage connection lifecycle. Plugins using this service
// must implement the WebSocketCallback capability to receive incoming messages and
// connection state changes.
//
//nd:hostservice name=WebSocket permission=websocket
type WebSocketService interface {
	// Connect establishes a WebSocket connection to the specified URL.
	//
	// Plugins that use this function must also implement the WebSocketCallback capability
	// to receive incoming messages and connection events.
	//
	// Parameters:
	//   - url: The WebSocket URL to connect to (ws:// or wss://)
	//   - headers: Optional HTTP headers to include in the handshake request
	//   - connectionID: Optional unique identifier for the connection. If empty, one will be generated
	//
	// Returns the connection ID that can be used to send messages or close the connection,
	// or an error if the connection fails.
	//nd:hostfunc
	Connect(ctx context.Context, url string, headers map[string]string, connectionID string) (newConnectionID string, err error)

	// SendText sends a text message over an established WebSocket connection.
	//
	// Parameters:
	//   - connectionID: The connection identifier returned by Connect
	//   - message: The text message to send
	//
	// Returns an error if the connection is not found or if sending fails.
	//nd:hostfunc
	SendText(ctx context.Context, connectionID, message string) error

	// SendBinary sends binary data over an established WebSocket connection.
	//
	// Parameters:
	//   - connectionID: The connection identifier returned by Connect
	//   - data: The binary data to send
	//
	// Returns an error if the connection is not found or if sending fails.
	//nd:hostfunc
	SendBinary(ctx context.Context, connectionID string, data []byte) error

	// CloseConnection gracefully closes a WebSocket connection.
	//
	// Parameters:
	//   - connectionID: The connection identifier returned by Connect
	//   - code: WebSocket close status code (e.g., 1000 for normal closure)
	//   - reason: Optional human-readable reason for closing
	//
	// Returns an error if the connection is not found or if closing fails.
	//nd:hostfunc
	CloseConnection(ctx context.Context, connectionID string, code int32, reason string) error
}
