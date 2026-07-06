package jellyfin

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/navidrome/navidrome/log"
)

// socketKeepAliveInterval is sent to the client in the initial ForceKeepAlive message, telling it
// how often (in seconds) it must send a KeepAlive, and used locally to bound the read deadline.
const socketKeepAliveInterval = 60

// socketReadTimeout is generous relative to socketKeepAliveInterval so a single delayed
// KeepAlive doesn't drop the connection.
const socketReadTimeout = 90 * time.Second

var socketUpgrader = websocket.Upgrader{
	// Finamp (and other Jellyfin clients) are not browsers, so there's no cross-origin risk
	// to guard against here; the connection is already authenticated via api_key.
	CheckOrigin: func(*http.Request) bool { return true },
}

// handleSocket implements Jellyfin's /socket WebSocket endpoint. Real-time clients like Finamp
// open this right after login; without it they 404-loop-reconnect instead of ever settling into
// a working session. This is a minimal implementation: it just keeps the connection alive and
// answers KeepAlive pings, with no session/playstate push, which is enough to stop the loop.
func (api *Router) handleSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := socketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warn(r.Context(), "Jellyfin API: WebSocket upgrade failed", err)
		return
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{"MessageType": "ForceKeepAlive", "Data": socketKeepAliveInterval}); err != nil {
		log.Warn(r.Context(), "Jellyfin API: WebSocket failed to send ForceKeepAlive", err)
		return
	}

	for {
		_ = conn.SetReadDeadline(time.Now().Add(socketReadTimeout))
		var msg struct {
			MessageType string `json:"MessageType"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg.MessageType == "KeepAlive" {
			if err := conn.WriteJSON(map[string]any{"MessageType": "KeepAlive"}); err != nil {
				return
			}
		}
	}
}
