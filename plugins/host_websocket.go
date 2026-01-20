package plugins

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/plugins/host"
)

// CapabilityWebSocket indicates the plugin can receive WebSocket callbacks.
// Detected when the plugin exports any of the WebSocket callback functions.
const CapabilityWebSocket Capability = "WebSocket"

// webSocketCallbackTimeout is the maximum duration allowed for a WebSocket callback.
const webSocketCallbackTimeout = 30 * time.Second

// WebSocket callback function names
const (
	FuncWebSocketOnTextMessage   = "nd_websocket_on_text_message"
	FuncWebSocketOnBinaryMessage = "nd_websocket_on_binary_message"
	FuncWebSocketOnError         = "nd_websocket_on_error"
	FuncWebSocketOnClose         = "nd_websocket_on_close"
)

func init() {
	registerCapability(
		CapabilityWebSocket,
		FuncWebSocketOnTextMessage,
		FuncWebSocketOnBinaryMessage,
		FuncWebSocketOnError,
		FuncWebSocketOnClose,
	)
}

// wsConnection represents an active WebSocket connection.
type wsConnection struct {
	conn     *websocket.Conn
	done     chan struct{}
	closeMu  sync.Mutex
	isClosed bool
}

// webSocketServiceImpl implements host.WebSocketService.
// It provides plugins with WebSocket communication capabilities.
type webSocketServiceImpl struct {
	pluginName    string
	manager       *Manager
	requiredHosts []string

	mu          sync.RWMutex
	connections map[string]*wsConnection
}

// newWebSocketService creates a new WebSocketService for a plugin.
func newWebSocketService(pluginName string, manager *Manager, permission *WebSocketPermission) *webSocketServiceImpl {
	return &webSocketServiceImpl{
		pluginName:    pluginName,
		manager:       manager,
		requiredHosts: permission.RequiredHosts,
		connections:   make(map[string]*wsConnection),
	}
}

func (s *webSocketServiceImpl) Connect(ctx context.Context, urlStr string, headers map[string]string, connectionID string) (string, error) {
	// Parse and validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Validate scheme
	if parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss" {
		return "", fmt.Errorf("invalid URL scheme: must be ws:// or wss://")
	}

	// Validate host against allowed hosts
	if !s.isHostAllowed(parsedURL.Host) {
		return "", fmt.Errorf("host %q is not allowed", parsedURL.Host)
	}

	// Generate connection ID if not provided
	if connectionID == "" {
		connectionID = id.NewRandom()
	}

	s.mu.Lock()
	if _, exists := s.connections[connectionID]; exists {
		s.mu.Unlock()
		return "", fmt.Errorf("connection ID %q already exists", connectionID)
	}
	s.mu.Unlock()

	// Create HTTP headers for handshake
	httpHeaders := http.Header{}
	for k, v := range headers {
		httpHeaders.Set(k, v)
	}

	// Establish WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, resp, err := dialer.DialContext(ctx, urlStr, httpHeaders)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}

	wsConn := &wsConnection{
		conn: conn,
		done: make(chan struct{}),
	}

	s.mu.Lock()
	s.connections[connectionID] = wsConn
	s.mu.Unlock()

	// Start read goroutine with manager's context.
	// We use manager.ctx instead of the caller's ctx because the readLoop must
	// outlive the Connect() call. The manager's context is cancelled during
	// application shutdown, ensuring graceful cleanup.
	go s.readLoop(s.manager.ctx, connectionID, wsConn)

	log.Debug(ctx, "WebSocket connected", "plugin", s.pluginName, "connectionID", connectionID, "url", urlStr)
	return connectionID, nil
}

func (s *webSocketServiceImpl) SendText(ctx context.Context, connectionID, message string) error {
	wsConn, err := s.getConnection(connectionID)
	if err != nil {
		return err
	}

	if err := wsConn.conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		return fmt.Errorf("failed to send text message: %w", err)
	}

	return nil
}

func (s *webSocketServiceImpl) SendBinary(ctx context.Context, connectionID string, data []byte) error {
	wsConn, err := s.getConnection(connectionID)
	if err != nil {
		return err
	}

	if err := wsConn.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("failed to send binary message: %w", err)
	}

	return nil
}

func (s *webSocketServiceImpl) CloseConnection(ctx context.Context, connectionID string, code int32, reason string) error {
	s.mu.Lock()
	wsConn, exists := s.connections[connectionID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("connection ID %q not found", connectionID)
	}
	delete(s.connections, connectionID)
	s.mu.Unlock()

	// Mark as closed to prevent callback
	wsConn.closeMu.Lock()
	wsConn.isClosed = true
	wsConn.closeMu.Unlock()

	// Send close message
	closeMsg := websocket.FormatCloseMessage(int(code), reason)
	_ = wsConn.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(5*time.Second))
	_ = wsConn.conn.Close()

	// Signal read goroutine to stop
	close(wsConn.done)

	// Invoke close callback
	s.invokeOnClose(ctx, connectionID, code, reason)

	log.Debug(ctx, "WebSocket connection closed", "plugin", s.pluginName, "connectionID", connectionID, "code", code)
	return nil
}

// Close closes all connections for this plugin.
// This is called when the plugin is unloaded.
func (s *webSocketServiceImpl) Close() error {
	s.mu.Lock()
	connections := make(map[string]*wsConnection, len(s.connections))
	for k, v := range s.connections {
		connections[k] = v
	}
	s.connections = make(map[string]*wsConnection)
	s.mu.Unlock()

	ctx := context.Background()
	for connID, wsConn := range connections {
		wsConn.closeMu.Lock()
		wsConn.isClosed = true
		wsConn.closeMu.Unlock()

		closeMsg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "plugin unloaded")
		err := wsConn.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(2*time.Second))
		if err != nil {
			log.Warn("Failed to send WebSocket close message on plugin unload", "plugin", s.pluginName, "connectionID", connID, "error", err)
		}
		err = wsConn.conn.Close()
		if err != nil {
			log.Warn("Failed to close WebSocket connection on plugin unload", "plugin", s.pluginName, "connectionID", connID, "error", err)
		}
		close(wsConn.done)

		s.invokeOnClose(ctx, connID, websocket.CloseGoingAway, "plugin unloaded")
		log.Debug("WebSocket connection closed on plugin unload", "plugin", s.pluginName, "connectionID", connID)
	}

	return nil
}

func (s *webSocketServiceImpl) getConnection(connectionID string) (*wsConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wsConn, exists := s.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection ID %q not found", connectionID)
	}
	return wsConn, nil
}

func (s *webSocketServiceImpl) isHostAllowed(host string) bool {
	// Strip port from host if present
	hostWithoutPort := host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		hostWithoutPort = host[:idx]
	}

	for _, pattern := range s.requiredHosts {
		if matchHostPattern(pattern, hostWithoutPort) {
			return true
		}
	}
	return false
}

// matchHostPattern matches a host against a pattern.
// Supports wildcards like *.example.com
func matchHostPattern(pattern, host string) bool {
	if pattern == host {
		return true
	}

	// Handle wildcard patterns like *.example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // Get .example.com
		return strings.HasSuffix(host, suffix)
	}

	return false
}

func (s *webSocketServiceImpl) readLoop(ctx context.Context, connectionID string, wsConn *wsConnection) {
	defer func() {
		// Remove connection if still present
		s.mu.Lock()
		delete(s.connections, connectionID)
		s.mu.Unlock()
	}()

	for {
		select {
		case <-wsConn.done:
			return
		default:
		}

		messageType, data, err := wsConn.conn.ReadMessage()
		if err != nil {
			wsConn.closeMu.Lock()
			isClosed := wsConn.isClosed
			wsConn.closeMu.Unlock()

			if isClosed {
				return
			}

			// Check if it's a close error
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				closeCode := websocket.CloseNoStatusReceived
				closeReason := ""
				var ce *websocket.CloseError
				if errors.As(err, &ce) {
					closeCode = ce.Code
					closeReason = ce.Text
				}
				s.invokeOnClose(ctx, connectionID, int32(closeCode), closeReason)
				return
			}

			// Other read error
			s.invokeOnError(ctx, connectionID, err.Error())
			return
		}

		switch messageType {
		case websocket.TextMessage:
			s.invokeOnTextMessage(ctx, connectionID, string(data))
		case websocket.BinaryMessage:
			s.invokeOnBinaryMessage(ctx, connectionID, data)
		}
	}
}

func (s *webSocketServiceImpl) invokeOnTextMessage(ctx context.Context, connectionID, message string) {
	instance := s.getPluginInstance()
	if instance == nil {
		return
	}

	input := capabilities.OnTextMessageRequest{
		ConnectionID: connectionID,
		Message:      message,
	}

	// Create a timeout context for this callback invocation
	callbackCtx, cancel := context.WithTimeout(ctx, webSocketCallbackTimeout)
	defer cancel()

	start := time.Now()
	err := callPluginFunctionNoOutput(callbackCtx, instance, FuncWebSocketOnTextMessage, input)
	if err != nil {
		// Don't log error if function simply doesn't exist (optional callback)
		if !errors.Is(errFunctionNotFound, err) {
			log.Error(ctx, "WebSocket text message callback failed", "plugin", s.pluginName, "connectionID", connectionID, "duration", time.Since(start), err)
		}
	}
}

func (s *webSocketServiceImpl) invokeOnBinaryMessage(ctx context.Context, connectionID string, data []byte) {
	instance := s.getPluginInstance()
	if instance == nil {
		return
	}

	input := capabilities.OnBinaryMessageRequest{
		ConnectionID: connectionID,
		Data:         base64.StdEncoding.EncodeToString(data),
	}

	// Create a timeout context for this callback invocation
	callbackCtx, cancel := context.WithTimeout(ctx, webSocketCallbackTimeout)
	defer cancel()

	start := time.Now()
	err := callPluginFunctionNoOutput(callbackCtx, instance, FuncWebSocketOnBinaryMessage, input)
	if err != nil {
		// Don't log error if function simply doesn't exist (optional callback)
		if !errors.Is(errFunctionNotFound, err) {
			log.Error(ctx, "WebSocket binary message callback failed", "plugin", s.pluginName, "connectionID", connectionID, "duration", time.Since(start), err)
		}
	}
}

func (s *webSocketServiceImpl) invokeOnError(ctx context.Context, connectionID, errorMsg string) {
	instance := s.getPluginInstance()
	if instance == nil {
		return
	}

	input := capabilities.OnErrorRequest{
		ConnectionID: connectionID,
		Error:        errorMsg,
	}

	// Create a timeout context for this callback invocation
	callbackCtx, cancel := context.WithTimeout(ctx, webSocketCallbackTimeout)
	defer cancel()

	start := time.Now()
	err := callPluginFunctionNoOutput(callbackCtx, instance, FuncWebSocketOnError, input)
	if err != nil {
		// Don't log error if function simply doesn't exist (optional callback)
		if !errors.Is(errFunctionNotFound, err) {
			log.Error(ctx, "WebSocket error callback failed", "plugin", s.pluginName, "connectionID", connectionID, "duration", time.Since(start), err)
		}
	}
}

func (s *webSocketServiceImpl) invokeOnClose(ctx context.Context, connectionID string, code int32, reason string) {
	instance := s.getPluginInstance()
	if instance == nil {
		return
	}

	input := capabilities.OnCloseRequest{
		ConnectionID: connectionID,
		Code:         code,
		Reason:       reason,
	}

	// Create a timeout context for this callback invocation
	callbackCtx, cancel := context.WithTimeout(ctx, webSocketCallbackTimeout)
	defer cancel()

	start := time.Now()
	err := callPluginFunctionNoOutput(callbackCtx, instance, FuncWebSocketOnClose, input)
	if err != nil {
		// Don't log error if function simply doesn't exist (optional callback)
		if !errors.Is(errFunctionNotFound, err) {
			log.Error(ctx, "WebSocket close callback failed", "plugin", s.pluginName, "connectionID", connectionID, "duration", time.Since(start), err)
		}
	}
}

func (s *webSocketServiceImpl) getPluginInstance() *plugin {
	s.manager.mu.RLock()
	instance, ok := s.manager.plugins[s.pluginName]
	s.manager.mu.RUnlock()

	if !ok {
		log.Warn("Plugin not loaded for WebSocket callback", "plugin", s.pluginName)
		return nil
	}

	return instance
}

// Verify interface implementation
var _ host.WebSocketService = (*webSocketServiceImpl)(nil)
