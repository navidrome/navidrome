package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gorillaws "github.com/gorilla/websocket"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/plugins/host/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocket Host Service", func() {
	var (
		wsService      *websocketService
		manager        *managerImpl
		ctx            context.Context
		server         *httptest.Server
		upgrader       gorillaws.Upgrader
		serverMessages []string
		serverMu       sync.Mutex
	)

	// WebSocket echo server handler
	echoHandler := func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if r.Header.Get("X-Test-Header") != "test-value" {
			http.Error(w, "Missing or invalid X-Test-Header", http.StatusBadRequest)
			return
		}

		// Upgrade connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Echo messages back
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			// Store the received message for verification
			if mt == gorillaws.TextMessage {
				msg := string(message)
				serverMu.Lock()
				serverMessages = append(serverMessages, msg)
				serverMu.Unlock()
			}

			// Echo it back
			err = conn.WriteMessage(mt, message)
			if err != nil {
				break
			}

			// If message is "close", close the connection
			if mt == gorillaws.TextMessage && string(message) == "close" {
				_ = conn.WriteControl(
					gorillaws.CloseMessage,
					gorillaws.FormatCloseMessage(gorillaws.CloseNormalClosure, "bye"),
					time.Now().Add(time.Second),
				)
				break
			}
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		serverMessages = make([]string, 0)
		serverMu = sync.Mutex{}

		// Create a test WebSocket server
		//upgrader = gorillaws.Upgrader{}
		server = httptest.NewServer(http.HandlerFunc(echoHandler))
		DeferCleanup(server.Close)

		// Create a new manager and websocket service
		manager = createManager(nil, metrics.NewNoopInstance())
		wsService = newWebsocketService(manager)
	})

	Describe("WebSocket operations", func() {
		var (
			pluginName   string
			connectionID string
			wsURL        string
		)

		BeforeEach(func() {
			pluginName = "test-plugin"
			connectionID = "test-connection-id"
			wsURL = "ws" + strings.TrimPrefix(server.URL, "http")
		})

		It("connects to a WebSocket server", func() {
			// Connect to the WebSocket server
			req := &websocket.ConnectRequest{
				Url: wsURL,
				Headers: map[string]string{
					"X-Test-Header": "test-value",
				},
				ConnectionId: connectionID,
			}

			resp, err := wsService.connect(ctx, pluginName, req, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ConnectionId).ToNot(BeEmpty())
			connectionID = resp.ConnectionId

			// Verify that the connection was added to the service
			internalID := pluginName + ":" + connectionID
			Expect(wsService.hasConnection(internalID)).To(BeTrue())
		})

		It("sends and receives text messages", func() {
			// Connect to the WebSocket server
			req := &websocket.ConnectRequest{
				Url: wsURL,
				Headers: map[string]string{
					"X-Test-Header": "test-value",
				},
				ConnectionId: connectionID,
			}

			resp, err := wsService.connect(ctx, pluginName, req, nil)
			Expect(err).ToNot(HaveOccurred())
			connectionID = resp.ConnectionId

			// Send a text message
			textReq := &websocket.SendTextRequest{
				ConnectionId: connectionID,
				Message:      "hello websocket",
			}

			_, err = wsService.sendText(ctx, pluginName, textReq)
			Expect(err).ToNot(HaveOccurred())

			// Wait a bit for the message to be processed
			Eventually(func() []string {
				serverMu.Lock()
				defer serverMu.Unlock()
				return serverMessages
			}, "1s").Should(ContainElement("hello websocket"))
		})

		It("closes a WebSocket connection", func() {
			// Connect to the WebSocket server
			req := &websocket.ConnectRequest{
				Url: wsURL,
				Headers: map[string]string{
					"X-Test-Header": "test-value",
				},
				ConnectionId: connectionID,
			}

			resp, err := wsService.connect(ctx, pluginName, req, nil)
			Expect(err).ToNot(HaveOccurred())
			connectionID = resp.ConnectionId

			initialCount := wsService.connectionCount()

			// Close the connection
			closeReq := &websocket.CloseRequest{
				ConnectionId: connectionID,
				Code:         1000, // Normal closure
				Reason:       "test complete",
			}

			_, err = wsService.close(ctx, pluginName, closeReq)
			Expect(err).ToNot(HaveOccurred())

			// Verify that the connection was removed
			Eventually(func() int {
				return wsService.connectionCount()
			}, "1s").Should(Equal(initialCount - 1))

			internalID := pluginName + ":" + connectionID
			Expect(wsService.hasConnection(internalID)).To(BeFalse())
		})

		It("handles connection errors gracefully", func() {
			if testing.Short() {
				GinkgoT().Skip("skipping test in short mode.")
			}

			// Try to connect to an invalid URL
			req := &websocket.ConnectRequest{
				Url:          "ws://invalid-url-that-does-not-exist",
				Headers:      map[string]string{},
				ConnectionId: connectionID,
			}

			_, err := wsService.connect(ctx, pluginName, req, nil)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when attempting to use non-existent connection", func() {
			// Try to send a message to a non-existent connection
			textReq := &websocket.SendTextRequest{
				ConnectionId: "non-existent-connection",
				Message:      "this should fail",
			}

			sendResp, err := wsService.sendText(ctx, pluginName, textReq)
			Expect(err).ToNot(HaveOccurred())
			Expect(sendResp.Error).To(ContainSubstring("connection not found"))

			// Try to close a non-existent connection
			closeReq := &websocket.CloseRequest{
				ConnectionId: "non-existent-connection",
				Code:         1000,
				Reason:       "test complete",
			}

			closeResp, err := wsService.close(ctx, pluginName, closeReq)
			Expect(err).ToNot(HaveOccurred())
			Expect(closeResp.Error).To(ContainSubstring("connection not found"))
		})
	})
})
