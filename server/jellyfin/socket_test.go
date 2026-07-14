package jellyfin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("handleSocket", func() {
	var api *Router

	BeforeEach(func() {
		api = &Router{}
	})

	// Jellyfin's real-time clients (e.g. Finamp) open a WebSocket right after login; without
	// a working handshake here they 404-loop-reconnect instead of settling into a session.
	It("upgrades the connection and sends ForceKeepAlive", func() {
		srv := httptest.NewServer(http.HandlerFunc(api.handleSocket))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		Expect(conn.SetReadDeadline(time.Now().Add(2 * time.Second))).To(Succeed())
		var msg map[string]any
		Expect(conn.ReadJSON(&msg)).To(Succeed())
		Expect(msg["MessageType"]).To(Equal("ForceKeepAlive"))
		Expect(msg["Data"]).To(BeNumerically("==", 60))
	})

	It("replies to a KeepAlive message with a KeepAlive of its own", func() {
		srv := httptest.NewServer(http.HandlerFunc(api.handleSocket))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		Expect(conn.SetReadDeadline(time.Now().Add(2 * time.Second))).To(Succeed())
		var handshake map[string]any
		Expect(conn.ReadJSON(&handshake)).To(Succeed())
		Expect(handshake["MessageType"]).To(Equal("ForceKeepAlive"))

		Expect(conn.WriteJSON(map[string]any{"MessageType": "KeepAlive"})).To(Succeed())

		Expect(conn.SetReadDeadline(time.Now().Add(2 * time.Second))).To(Succeed())
		var reply map[string]any
		Expect(conn.ReadJSON(&reply)).To(Succeed())
		Expect(reply["MessageType"]).To(Equal("KeepAlive"))
	})

	It("closes the connection when the client disconnects, without leaving the handler hanging", func() {
		srv := httptest.NewServer(http.HandlerFunc(api.handleSocket))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(conn.SetReadDeadline(time.Now().Add(2 * time.Second))).To(Succeed())
		var handshake map[string]any
		Expect(conn.ReadJSON(&handshake)).To(Succeed())

		Expect(conn.Close()).To(Succeed())
	})

	// End-to-end: proves /socket is reachable through the full router (case-insensitive
	// wrapper + chi mux + auth middleware) with a real network listener, exactly as Finamp
	// hits it in production with ?api_key=<jwt>.
	Context("mounted behind the full router and auth middleware", func() {
		var ds *tests.MockDataStore
		var token string

		BeforeEach(func() {
			ds = &tests.MockDataStore{}
			auth.Init(ds)
			ur := ds.User(context.Background()).(*tests.MockedUserRepo)
			Expect(ur.Put(&model.User{ID: "u1", UserName: "alice", NewPassword: "secret"})).To(Succeed())

			t, err := auth.CreateToken(&model.User{ID: "u1", UserName: "alice"})
			Expect(err).ToNot(HaveOccurred())
			token = t

			api = New(ds, nil, nil, nil, nil, nil, nil, nil)
		})

		It("upgrades when authenticated via the api_key query parameter", func() {
			srv := httptest.NewServer(api)
			defer srv.Close()

			wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/socket?api_key=" + token
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			Expect(err).ToNot(HaveOccurred())
			defer conn.Close()

			Expect(conn.SetReadDeadline(time.Now().Add(2 * time.Second))).To(Succeed())
			var msg map[string]any
			Expect(conn.ReadJSON(&msg)).To(Succeed())
			Expect(msg["MessageType"]).To(Equal("ForceKeepAlive"))
		})

		It("rejects the upgrade with no api_key", func() {
			srv := httptest.NewServer(api)
			defer srv.Close()

			wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/socket"
			_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
			Expect(err).To(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})
	})
})
