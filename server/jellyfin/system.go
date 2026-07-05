package jellyfin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// jellyfinVersion is the Jellyfin API version we advertise in the handshake. Clients parse
// Version as a Jellyfin release and feature-gate on it, so it must stay a real Jellyfin
// version rather than Navidrome's own.
const jellyfinVersion = "10.8.13"

func (api *Router) serverName() string {
	if conf.Server.Jellyfin.ServerName != "" {
		return conf.Server.Jellyfin.ServerName
	}
	return fmt.Sprintf("Navidrome %s", consts.Version)
}

// serverID returns a stable Id that survives restarts, get-or-created in the Property table
// the same way core/metrics/insights.go derives its InsightsID. Jellyfin clients cache
// ServerId across sessions, so a per-process value would break re-authentication.
// api.ds is nil only in unit tests that construct Router{} directly; New() always sets it.
//
// Resolution happens once per Router (sync.Once), so concurrent first-boot requests can't
// each generate and persist a different UUID before the row settles.
func (api *Router) serverID(ctx context.Context) string {
	api.serverIDOnce.Do(func() {
		if api.ds == nil {
			api.serverIDVal = uuid.NewString()
			return
		}
		id, err := api.ds.Property(ctx).Get(consts.JellyfinServerIDKey)
		if err != nil {
			id = uuid.NewString()
			if err := api.ds.Property(ctx).Put(consts.JellyfinServerIDKey, id); err != nil {
				log.Error(ctx, "Jellyfin API: could not persist server id", err)
			}
		}
		api.serverIDVal = id
	})
	return api.serverIDVal
}

func (api *Router) publicInfo(ctx context.Context) dto.PublicSystemInfo {
	return dto.PublicSystemInfo{
		ServerName:             api.serverName(),
		Version:                jellyfinVersion,
		ProductName:            "Jellyfin Server",
		Id:                     api.serverID(ctx),
		StartupWizardCompleted: true,
	}
}

func (api *Router) getPublicSystemInfo(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, api.publicInfo(r.Context()))
}

// ping answers /System/Ping with a bare plain-text server name, not a JSON-quoted string:
// Jellyfin's own server does this, and clients parse the raw body.
func (api *Router) ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(api.serverName()))
}

func (api *Router) quickConnectEnabled(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, false)
}
