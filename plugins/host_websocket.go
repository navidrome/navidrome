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
	ws         *websocketService
	pluginName string
}

func (s WebSocketHostFunctions) Connect(ctx context.Context, req *websocket.ConnectRequest) (*websocket.ConnectResponse, error) {
	return s.ws.connect(ctx, s.pluginName, req)
}

func (s WebSocketHostFunctions) SendText(ctx context.Context, req *websocket.SendTextRequest) (*websocket.SendTextResponse, error) {
	return s.ws.sendText(ctx, s.pluginName, req)
}

func (s WebSocketHostFunctions) SendBinary(ctx context.Context, req *websocket.SendBinaryRequest) (*websocket.SendBinaryResponse, error) {
	return s.ws.sendBinary(ctx, s.pluginName, req)
}

func (s WebSocketHostFunctions) Close(ctx context.Context, req *websocket.CloseRequest) (*websocket.CloseResponse, error) {
	return s.ws.close(ctx, s.pluginName, req)
}

// websocketService implements the WebSocket service functionality
type websocketService struct {
	connections map[string]*WebSocketConnection
	manager     *Manager
	mu          sync.RWMutex
}

// newWebsocketService creates a new websocketService instance
func newWebsocketService(manager *Manager) *websocketService {
	return &websocketService{
		connections: make(map[string]*WebSocketConnection),
		manager:     manager,
	}
}

// HostFunctions returns the WebSocketHostFunctions for the given plugin
func (s *websocketService) HostFunctions(pluginName string) WebSocketHostFunctions {
	return WebSocketHostFunctions{
		ws:         s,
		pluginName: pluginName,
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

// connect establishes a new WebSocket connection
func (s *websocketService) connect(ctx context.Context, pluginName string, req *websocket.ConnectRequest) (*websocket.ConnectResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("websocket service not properly initialized")
	}

	// Create websocket dialer with the headers
	dialer := gorillaws.DefaultDialer
	header := make(map[string][]string)
	for k, v := range req.Headers {
		header[k] = []string{v}
	}

	// Connect to the WebSocket server
	conn, _, err := dialer.DialContext(ctx, req.Url, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}

	// Generate a connection ID
	if req.ConnectionId == "" {
		req.ConnectionId, _ = gonanoid.New(10)
	}
	connectionID := req.ConnectionId
	internalConnectionID := pluginName + ":" + connectionID

	// Create the connection object
	wsConn := &WebSocketConnection{
		Conn:         conn,
		PluginName:   pluginName,
		ConnectionID: connectionID,
		Done:         make(chan struct{}),
	}

	// Store the connection
	s.mu.Lock()
	s.connections[internalConnectionID] = wsConn
	s.mu.Unlock()

	log.Debug("WebSocket connection established", "plugin", pluginName, "connectionID", connectionID, "url", req.Url)

	// Start the message handling goroutine
	go s.handleMessages(internalConnectionID, wsConn)

	return &websocket.ConnectResponse{
		ConnectionId: connectionID,
	}, nil
}

// sendText sends a text message over a WebSocket connection
func (s *websocketService) sendText(_ context.Context, pluginName string, req *websocket.SendTextRequest) (*websocket.SendTextResponse, error) {
	internalConnectionID := pluginName + ":" + req.ConnectionId

	s.mu.RLock()
	conn, exists := s.connections[internalConnectionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("connection not found")
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if err := conn.Conn.WriteMessage(gorillaws.TextMessage, []byte(req.Message)); err != nil {
		return nil, fmt.Errorf("failed to send text message: %w", err)
	}

	return &websocket.SendTextResponse{}, nil
}

// sendBinary sends binary data over a WebSocket connection
func (s *websocketService) sendBinary(_ context.Context, pluginName string, req *websocket.SendBinaryRequest) (*websocket.SendBinaryResponse, error) {
	internalConnectionID := pluginName + ":" + req.ConnectionId

	s.mu.RLock()
	conn, exists := s.connections[internalConnectionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("connection not found")
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if err := conn.Conn.WriteMessage(gorillaws.BinaryMessage, req.Data); err != nil {
		return nil, fmt.Errorf("failed to send binary message: %w", err)
	}

	return &websocket.SendBinaryResponse{}, nil
}

// close closes a WebSocket connection
func (s *websocketService) close(_ context.Context, pluginName string, req *websocket.CloseRequest) (*websocket.CloseResponse, error) {
	internalConnectionID := pluginName + ":" + req.ConnectionId

	s.mu.Lock()
	conn, exists := s.connections[internalConnectionID]
	if !exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("connection not found")
	}
	delete(s.connections, internalConnectionID)
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
		log.Error("Error sending close message", "plugin", pluginName, "error", err)
	}

	if err := conn.Conn.Close(); err != nil {
		return nil, fmt.Errorf("error closing connection: %w", err)
	}

	log.Debug("WebSocket connection closed", "plugin", pluginName, "connectionID", req.ConnectionId)
	return &websocket.CloseResponse{}, nil
}

// handleMessages processes incoming WebSocket messages
func (s *websocketService) handleMessages(internalConnectionID string, conn *WebSocketConnection) {
	// Get the original connection ID (without plugin prefix)
	parts := strings.Split(internalConnectionID, ":")
	if len(parts) != 2 {
		log.Error("Invalid internal connection ID format", "id", internalConnectionID)
		return
	}
	connectionID := parts[1]

	defer func() {
		// Ensure connection is removed from the map if not already removed
		s.mu.Lock()
		delete(s.connections, internalConnectionID)
		s.mu.Unlock()

		log.Debug("WebSocket message handler stopped", "plugin", conn.PluginName, "connectionID", connectionID)
	}()

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
				s.notifyError(connectionID, conn, err.Error())
				return
			}

			// Reset the read deadline
			_ = conn.Conn.SetReadDeadline(time.Time{})

			// Process the message based on its type
			switch messageType {
			case gorillaws.TextMessage:
				s.notifyTextMessage(connectionID, conn, string(message))
			case gorillaws.BinaryMessage:
				s.notifyBinaryMessage(connectionID, conn, message)
			case gorillaws.CloseMessage:
				code := gorillaws.CloseNormalClosure
				reason := ""
				if len(message) >= 2 {
					code = int(binary.BigEndian.Uint16(message[:2]))
					if len(message) > 2 {
						reason = string(message[2:])
					}
				}
				s.notifyClose(connectionID, conn, code, reason)
				return
			}
		}
	}
}

// notifyTextMessage notifies the plugin of a text message
func (s *websocketService) notifyTextMessage(connectionID string, conn *WebSocketConnection, message string) {
	log.Debug("WebSocket text message received", "plugin", conn.PluginName, "connectionID", connectionID)
	start := time.Now()

	// Create request
	req := &api.OnTextMessageRequest{
		ConnectionId: connectionID,
		Message:      message,
	}

	// Get the plugin
	p := s.manager.LoadPlugin(conn.PluginName, CapabilityWebSocketCallback)
	if p == nil {
		log.Error("Plugin not found for WebSocket callback", "plugin", conn.PluginName)
		return
	}

	// Get instance
	ctx := context.Background()
	inst, closeFn, err := p.Instantiate(ctx)
	if err != nil {
		log.Error("Error getting plugin instance for WebSocket callback", "plugin", conn.PluginName, err)
		return
	}
	defer closeFn()

	// Type-check the plugin
	plugin, ok := inst.(api.WebSocketCallback)
	if !ok {
		log.Error("Plugin does not implement WebSocketCallback", "plugin", conn.PluginName)
		return
	}

	// Call the plugin's OnTextMessage method
	log.Trace(ctx, "Executing WebSocket text message callback", "plugin", conn.PluginName, "connectionID", connectionID)
	_, err = plugin.OnTextMessage(ctx, req)
	if err != nil {
		log.Error("Error executing WebSocket text message callback", "plugin", conn.PluginName, "elapsed", time.Since(start), err)
		return
	}
	log.Debug("WebSocket text message callback executed", "plugin", conn.PluginName, "elapsed", time.Since(start))
}

// notifyBinaryMessage notifies the plugin of a binary message
func (s *websocketService) notifyBinaryMessage(connectionID string, conn *WebSocketConnection, data []byte) {
	if conn.ConnectionID == "" {
		log.Warn("No callback ID registered for binary message", "plugin", conn.PluginName)
		return
	}

	log.Debug("WebSocket binary message received", "plugin", conn.PluginName, "connectionID", connectionID, "size", len(data))
	start := time.Now()

	// Create request
	req := &api.OnBinaryMessageRequest{
		ConnectionId: connectionID,
		Data:         data,
	}

	// Get the plugin
	p := s.manager.LoadPlugin(conn.PluginName, CapabilityWebSocketCallback)
	if p == nil {
		log.Error("Plugin not found for WebSocket callback", "plugin", conn.PluginName)
		return
	}

	// Get instance
	ctx := context.Background()
	inst, closeFn, err := p.Instantiate(ctx)
	if err != nil {
		log.Error("Error getting plugin instance for WebSocket callback", "plugin", conn.PluginName, err)
		return
	}
	defer closeFn()

	// Type-check the plugin
	plugin, ok := inst.(api.WebSocketCallback)
	if !ok {
		log.Error("Plugin does not implement WebSocketCallback", "plugin", conn.PluginName)
		return
	}

	// Call the plugin's OnBinaryMessage method
	log.Trace(ctx, "Executing WebSocket binary message callback", "plugin", conn.PluginName, "connectionID", connectionID)
	_, err = plugin.OnBinaryMessage(ctx, req)
	if err != nil {
		log.Error("Error executing WebSocket binary message callback", "plugin", conn.PluginName, "elapsed", time.Since(start), err)
		return
	}
	log.Debug("WebSocket binary message callback executed", "plugin", conn.PluginName, "elapsed", time.Since(start))
}

// notifyError notifies the plugin of an error
func (s *websocketService) notifyError(connectionID string, conn *WebSocketConnection, errorMsg string) {
	if conn.ConnectionID == "" {
		log.Warn("No callback ID registered for error", "plugin", conn.PluginName)
		return
	}

	log.Debug("WebSocket error occurred", "plugin", conn.PluginName, "connectionID", connectionID, "error", errorMsg)
	start := time.Now()

	// Create request
	req := &api.OnErrorRequest{
		ConnectionId: connectionID,
		Error:        errorMsg,
	}

	// Get the plugin
	p := s.manager.LoadPlugin(conn.PluginName, CapabilityWebSocketCallback)
	if p == nil {
		log.Error("Plugin not found for WebSocket callback", "plugin", conn.PluginName)
		return
	}

	// Get instance
	ctx := context.Background()
	inst, closeFn, err := p.Instantiate(ctx)
	if err != nil {
		log.Error("Error getting plugin instance for WebSocket callback", "plugin", conn.PluginName, err)
		return
	}
	defer closeFn()

	// Type-check the plugin
	plugin, ok := inst.(api.WebSocketCallback)
	if !ok {
		log.Error("Plugin does not implement WebSocketCallback", "plugin", conn.PluginName)
		return
	}

	// Call the plugin's OnError method
	log.Trace(ctx, "Executing WebSocket error callback", "plugin", conn.PluginName, "connectionID", connectionID)
	_, err = plugin.OnError(ctx, req)
	if err != nil {
		log.Error("Error executing WebSocket error callback", "plugin", conn.PluginName, "elapsed", time.Since(start), err)
		return
	}
	log.Debug("WebSocket error callback executed", "plugin", conn.PluginName, "elapsed", time.Since(start))
}

// notifyClose notifies the plugin that the connection was closed
func (s *websocketService) notifyClose(connectionID string, conn *WebSocketConnection, code int, reason string) {
	if conn.ConnectionID == "" {
		log.Warn("No callback ID registered for close", "plugin", conn.PluginName)
		return
	}

	log.Debug("WebSocket connection was closed", "plugin", conn.PluginName, "connectionID", connectionID, "code", code, "reason", reason)
	start := time.Now()

	// Create request
	req := &api.OnCloseRequest{
		ConnectionId: connectionID,
		Code:         int32(code),
		Reason:       reason,
	}

	// Get the plugin
	p := s.manager.LoadPlugin(conn.PluginName, CapabilityWebSocketCallback)
	if p == nil {
		log.Error("Plugin not found for WebSocket callback", "plugin", conn.PluginName)
		return
	}

	// Get instance
	ctx := context.Background()
	inst, closeFn, err := p.Instantiate(ctx)
	if err != nil {
		log.Error("Error getting plugin instance for WebSocket callback", "plugin", conn.PluginName, err)
		return
	}
	defer closeFn()

	// Type-check the plugin
	plugin, ok := inst.(api.WebSocketCallback)
	if !ok {
		log.Error("Plugin does not implement WebSocketCallback", "plugin", conn.PluginName)
		return
	}

	// Call the plugin's OnClose method
	log.Trace(ctx, "Executing WebSocket close callback", "plugin", conn.PluginName, "connectionID", connectionID)
	_, err = plugin.OnClose(ctx, req)
	if err != nil {
		log.Error("Error executing WebSocket close callback", "plugin", conn.PluginName, "elapsed", time.Since(start), err)
		return
	}
	log.Debug("WebSocket close callback executed", "plugin", conn.PluginName, "elapsed", time.Since(start))
}
