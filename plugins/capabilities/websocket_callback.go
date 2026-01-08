package capabilities

// WebSocketCallback provides WebSocket message handling.
// This capability allows plugins to receive callbacks for WebSocket events
// such as text messages, binary messages, errors, and connection closures.
// Plugins that use the WebSocket host service must implement this capability
// to handle incoming events.
//
//nd:capability name=websocket
type WebSocketCallback interface {
	// OnTextMessage is called when a text message is received on a WebSocket connection.
	//nd:export name=nd_websocket_on_text_message
	OnTextMessage(OnTextMessageRequest) error

	// OnBinaryMessage is called when a binary message is received on a WebSocket connection.
	//nd:export name=nd_websocket_on_binary_message
	OnBinaryMessage(OnBinaryMessageRequest) error

	// OnError is called when an error occurs on a WebSocket connection.
	//nd:export name=nd_websocket_on_error
	OnError(OnErrorRequest) error

	// OnClose is called when a WebSocket connection is closed.
	//nd:export name=nd_websocket_on_close
	OnClose(OnCloseRequest) error
}

// OnTextMessageRequest is the request provided when a text message is received.
type OnTextMessageRequest struct {
	// ConnectionID is the unique identifier for the WebSocket connection that received the message.
	ConnectionID string `json:"connectionId"`
	// Message is the text message content received from the WebSocket.
	Message string `json:"message"`
}

// OnBinaryMessageRequest is the request provided when a binary message is received.
type OnBinaryMessageRequest struct {
	// ConnectionID is the unique identifier for the WebSocket connection that received the message.
	ConnectionID string `json:"connectionId"`
	// Data is the binary data received from the WebSocket, encoded as base64.
	Data string `json:"data"`
}

// OnErrorRequest is the request provided when an error occurs on a WebSocket connection.
type OnErrorRequest struct {
	// ConnectionID is the unique identifier for the WebSocket connection where the error occurred.
	ConnectionID string `json:"connectionId"`
	// Error is the error message describing what went wrong.
	Error string `json:"error"`
}

// OnCloseRequest is the request provided when a WebSocket connection is closed.
type OnCloseRequest struct {
	// ConnectionID is the unique identifier for the WebSocket connection that was closed.
	ConnectionID string `json:"connectionId"`
	// Code is the WebSocket close status code (e.g., 1000 for normal closure,
	// 1001 for going away, 1006 for abnormal closure).
	Code int32 `json:"code"`
	// Reason is the human-readable reason for the connection closure, if provided.
	Reason string `json:"reason"`
}
