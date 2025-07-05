package plugins

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	gorillaws "github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/websocket"
)

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	Conn         *gorillaws.Conn
	PluginName   string
	ConnectionID string
	Done         chan struct{}
	mu           sync.Mutex
}

// WebSocketHostFunctions implements the websocket.WebSocketService interface
type WebSocketHostFunctions struct {
	ws          *websocketService
	pluginID    string
	permissions *webSocketPermissions
}

func (s WebSocketHostFunctions) Connect(ctx context.Context, req *websocket.ConnectRequest) (*websocket.ConnectResponse, error) {
	return s.ws.connect(ctx, s.pluginID, req, s.permissions)
}

func (s WebSocketHostFunctions) SendText(ctx context.Context, req *websocket.SendTextRequest) (*websocket.SendTextResponse, error) {
	return s.ws.sendText(ctx, s.pluginID, req)
}

func (s WebSocketHostFunctions) SendBinary(ctx context.Context, req *websocket.SendBinaryRequest) (*websocket.SendBinaryResponse, error) {
	return s.ws.sendBinary(ctx, s.pluginID, req)
}

func (s WebSocketHostFunctions) Close(ctx context.Context, req *websocket.CloseRequest) (*websocket.CloseResponse, error) {
	return s.ws.close(ctx, s.pluginID, req)
}

// websocketService implements the WebSocket service functionality
type websocketService struct {
	connections map[string]*WebSocketConnection
	manager     *managerImpl
	mu          sync.RWMutex
}

// newWebsocketService creates a new websocketService instance
func newWebsocketService(manager *managerImpl) *websocketService {
	return &websocketService{
		connections: make(map[string]*WebSocketConnection),
		manager:     manager,
	}
}

// HostFunctions returns the WebSocketHostFunctions for the given plugin
func (s *websocketService) HostFunctions(pluginID string, permissions *webSocketPermissions) WebSocketHostFunctions {
	return WebSocketHostFunctions{
		ws:          s,
		pluginID:    pluginID,
		permissions: permissions,
	}
}

// Safe accessor methods

// hasConnection safely checks if a connection exists
func (s *websocketService) hasConnection(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.connections[id]
	return exists
}

// connectionCount safely returns the number of connections
func (s *websocketService) connectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// getConnection safely retrieves a connection by internal ID
func (s *websocketService) getConnection(internalConnectionID string) (*WebSocketConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, exists := s.connections[internalConnectionID]

	if !exists {
		return nil, fmt.Errorf("connection not found")
	}
	return conn, nil
}

// internalConnectionID builds the internal connection ID from plugin and connection ID
func internalConnectionID(pluginName, connectionID string) string {
	return pluginName + ":" + connectionID
}

// extractConnectionID extracts the original connection ID from an internal ID
func extractConnectionID(internalID string) (string, error) {
	parts := strings.Split(internalID, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid internal connection ID format: %s", internalID)
	}
	return parts[1], nil
}

// connect establishes a new WebSocket connection
func (s *websocketService) connect(ctx context.Context, pluginID string, req *websocket.ConnectRequest, permissions *webSocketPermissions) (*websocket.ConnectResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("websocket service not properly initialized")
	}

	// Check permissions if they exist
	if permissions != nil {
		if err := permissions.IsConnectionAllowed(req.Url); err != nil {
			log.Warn(ctx, "WebSocket connection blocked by permissions", "plugin", pluginID, "url", req.Url, err)
			return &websocket.ConnectResponse{Error: "Connection blocked by plugin permissions: " + err.Error()}, nil
		}
	}

	// Create websocket dialer with the headers
	dialer := gorillaws.DefaultDialer
	header := make(map[string][]string)
	for k, v := range req.Headers {
		header[k] = []string{v}
	}

	// Connect to the WebSocket server
	conn, resp, err := dialer.DialContext(ctx, req.Url, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}
	defer resp.Body.Close()

	// Generate a connection ID
	if req.ConnectionId == "" {
		req.ConnectionId, _ = gonanoid.New(10)
	}
	connectionID := req.ConnectionId
	internal := internalConnectionID(pluginID, connectionID)

	// Create the connection object
	wsConn := &WebSocketConnection{
		Conn:         conn,
		PluginName:   pluginID,
		ConnectionID: connectionID,
		Done:         make(chan struct{}),
	}

	// Store the connection
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections[internal] = wsConn

	log.Debug("WebSocket connection established", "plugin", pluginID, "connectionID", connectionID, "url", req.Url)

	// Start the message handling goroutine
	go s.handleMessages(internal, wsConn)

	return &websocket.ConnectResponse{
		ConnectionId: connectionID,
	}, nil
}

// writeMessage is a helper to send messages to a websocket connection
func (s *websocketService) writeMessage(pluginID string, connID string, messageType int, data []byte) error {
	internal := internalConnectionID(pluginID, connID)

	conn, err := s.getConnection(internal)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if err := conn.Conn.WriteMessage(messageType, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// sendText sends a text message over a WebSocket connection
func (s *websocketService) sendText(ctx context.Context, pluginID string, req *websocket.SendTextRequest) (*websocket.SendTextResponse, error) {
	if err := s.writeMessage(pluginID, req.ConnectionId, gorillaws.TextMessage, []byte(req.Message)); err != nil {
		return &websocket.SendTextResponse{Error: err.Error()}, nil //nolint:nilerr
	}
	return &websocket.SendTextResponse{}, nil
}

// sendBinary sends binary data over a WebSocket connection
func (s *websocketService) sendBinary(ctx context.Context, pluginID string, req *websocket.SendBinaryRequest) (*websocket.SendBinaryResponse, error) {
	if err := s.writeMessage(pluginID, req.ConnectionId, gorillaws.BinaryMessage, req.Data); err != nil {
		return &websocket.SendBinaryResponse{Error: err.Error()}, nil //nolint:nilerr
	}
	return &websocket.SendBinaryResponse{}, nil
}

// close closes a WebSocket connection
func (s *websocketService) close(ctx context.Context, pluginID string, req *websocket.CloseRequest) (*websocket.CloseResponse, error) {
	internal := internalConnectionID(pluginID, req.ConnectionId)

	s.mu.Lock()
	conn, exists := s.connections[internal]
	if !exists {
		s.mu.Unlock()
		return &websocket.CloseResponse{Error: "connection not found"}, nil
	}
	delete(s.connections, internal)
	s.mu.Unlock()

	// Signal the message handling goroutine to stop
	close(conn.Done)

	// Close the connection with the specified code and reason
	conn.mu.Lock()
	defer conn.mu.Unlock()

	err := conn.Conn.WriteControl(
		gorillaws.CloseMessage,
		gorillaws.FormatCloseMessage(int(req.Code), req.Reason),
		time.Now().Add(time.Second),
	)
	if err != nil {
		log.Error("Error sending close message", "plugin", pluginID, "error", err)
	}

	if err := conn.Conn.Close(); err != nil {
		return nil, fmt.Errorf("error closing connection: %w", err)
	}

	log.Debug("WebSocket connection closed", "plugin", pluginID, "connectionID", req.ConnectionId)
	return &websocket.CloseResponse{}, nil
}

// handleMessages processes incoming WebSocket messages
func (s *websocketService) handleMessages(internalID string, conn *WebSocketConnection) {
	// Get the original connection ID (without plugin prefix)
	connectionID, err := extractConnectionID(internalID)
	if err != nil {
		log.Error("Invalid internal connection ID", "id", internalID, "error", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		// Ensure the connection is removed from the map if not already removed
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.connections, internalID)

		log.Debug("WebSocket message handler stopped", "plugin", conn.PluginName, "connectionID", connectionID)
	}()

	// Add connection info to context
	ctx = log.NewContext(ctx,
		"connectionID", connectionID,
		"plugin", conn.PluginName,
	)

	for {
		select {
		case <-conn.Done:
			// Connection was closed by a Close call
			return
		default:
			// Set a read deadline
			_ = conn.Conn.SetReadDeadline(time.Now().Add(time.Second * 60))

			// Read the next message
			messageType, message, err := conn.Conn.ReadMessage()
			if err != nil {
				s.notifyErrorCallback(ctx, connectionID, conn, err.Error())
				return
			}

			// Reset the read deadline
			_ = conn.Conn.SetReadDeadline(time.Time{})

			// Process the message based on its type
			switch messageType {
			case gorillaws.TextMessage:
				s.notifyTextCallback(ctx, connectionID, conn, string(message))
			case gorillaws.BinaryMessage:
				s.notifyBinaryCallback(ctx, connectionID, conn, message)
			case gorillaws.CloseMessage:
				code := gorillaws.CloseNormalClosure
				reason := ""
				if len(message) >= 2 {
					code = int(binary.BigEndian.Uint16(message[:2]))
					if len(message) > 2 {
						reason = string(message[2:])
					}
				}
				s.notifyCloseCallback(ctx, connectionID, conn, code, reason)
				return
			}
		}
	}
}

// executeCallback is a common function that handles the plugin loading and execution
// for all types of callbacks
func (s *websocketService) executeCallback(ctx context.Context, pluginID, methodName string, fn func(context.Context, api.WebSocketCallback) error) {
	log.Debug(ctx, "WebSocket received")

	start := time.Now()

	// Get the plugin
	p := s.manager.LoadPlugin(pluginID, CapabilityWebSocketCallback)
	if p == nil {
		log.Error(ctx, "Plugin not found for WebSocket callback")
		return
	}

	_, _ = callMethod(ctx, p, methodName, func(inst api.WebSocketCallback) (struct{}, error) {
		// Call the appropriate callback function
		log.Trace(ctx, "Executing WebSocket callback")
		if err := fn(ctx, inst); err != nil {
			log.Error(ctx, "Error executing WebSocket callback", "elapsed", time.Since(start), err)
			return struct{}{}, fmt.Errorf("error executing WebSocket callback: %w", err)
		}
		log.Debug(ctx, "WebSocket callback executed", "elapsed", time.Since(start))
		return struct{}{}, nil
	})
}

// notifyTextCallback notifies the plugin of a text message
func (s *websocketService) notifyTextCallback(ctx context.Context, connectionID string, conn *WebSocketConnection, message string) {
	req := &api.OnTextMessageRequest{
		ConnectionId: connectionID,
		Message:      message,
	}

	ctx = log.NewContext(ctx, "callback", "OnTextMessage", "size", len(message))

	s.executeCallback(ctx, conn.PluginName, "OnTextMessage", func(ctx context.Context, plugin api.WebSocketCallback) error {
		_, err := checkErr(plugin.OnTextMessage(ctx, req))
		return err
	})
}

// notifyBinaryCallback notifies the plugin of a binary message
func (s *websocketService) notifyBinaryCallback(ctx context.Context, connectionID string, conn *WebSocketConnection, data []byte) {
	req := &api.OnBinaryMessageRequest{
		ConnectionId: connectionID,
		Data:         data,
	}

	ctx = log.NewContext(ctx, "callback", "OnBinaryMessage", "size", len(data))

	s.executeCallback(ctx, conn.PluginName, "OnBinaryMessage", func(ctx context.Context, plugin api.WebSocketCallback) error {
		_, err := checkErr(plugin.OnBinaryMessage(ctx, req))
		return err
	})
}

// notifyErrorCallback notifies the plugin of an error
func (s *websocketService) notifyErrorCallback(ctx context.Context, connectionID string, conn *WebSocketConnection, errorMsg string) {
	req := &api.OnErrorRequest{
		ConnectionId: connectionID,
		Error:        errorMsg,
	}

	ctx = log.NewContext(ctx, "callback", "OnError", "error", errorMsg)

	s.executeCallback(ctx, conn.PluginName, "OnError", func(ctx context.Context, plugin api.WebSocketCallback) error {
		_, err := checkErr(plugin.OnError(ctx, req))
		return err
	})
}

// notifyCloseCallback notifies the plugin that the connection was closed
func (s *websocketService) notifyCloseCallback(ctx context.Context, connectionID string, conn *WebSocketConnection, code int, reason string) {
	req := &api.OnCloseRequest{
		ConnectionId: connectionID,
		Code:         int32(code),
		Reason:       reason,
	}

	ctx = log.NewContext(ctx, "callback", "OnClose", "code", code, "reason", reason)

	s.executeCallback(ctx, conn.PluginName, "OnClose", func(ctx context.Context, plugin api.WebSocketCallback) error {
		_, err := checkErr(plugin.OnClose(ctx, req))
		return err
	})
}
