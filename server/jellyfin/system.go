package jellyfin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// jellyfinVersion is the Jellyfin API version advertised in the handshake. Clients feature-gate
// on it, so it must stay a real Jellyfin release, not Navidrome's own version. 10.9+ is required
// for Feishin to use the server lyrics endpoint.
const jellyfinVersion = "10.9.11"

func (api *Router) serverName() string {
	if conf.Server.Jellyfin.ServerName != "" {
		return conf.Server.Jellyfin.ServerName
	}
	return fmt.Sprintf("Navidrome %s", consts.Version)
}

// serverID returns a stable Id that survives restarts, get-or-created in the Property table.
// Jellyfin clients cache ServerId across sessions, so a per-process value would break
// re-authentication. api.ds is nil only in unit tests; New() always sets it.
//
// The mutex serializes first-boot resolution so concurrent requests can't persist different
// UUIDs. Only a successful read or persisted id is cached; a transient failure yields a
// temporary id and retries on the next request rather than pinning a value.
func (api *Router) serverID(ctx context.Context) string {
	api.serverIDMu.Lock()
	defer api.serverIDMu.Unlock()
	if api.serverIDVal != "" {
		return api.serverIDVal
	}
	if api.ds == nil {
		api.serverIDVal = uuid.NewString()
		return api.serverIDVal
	}
	id, err := api.ds.Property(ctx).Get(consts.JellyfinServerIDKey)
	switch {
	case errors.Is(err, model.ErrNotFound):
		id = uuid.NewString()
		if err := api.ds.Property(ctx).Put(consts.JellyfinServerIDKey, id); err != nil {
			log.Error(ctx, "Jellyfin API: could not persist server id", err)
			return id
		}
	case err != nil:
		log.Error(ctx, "Jellyfin API: could not read server id", err)
		return uuid.NewString()
	}
	api.serverIDVal = id
	return api.serverIDVal
}

func (api *Router) publicInfo(r *http.Request) dto.PublicSystemInfo {
	return dto.PublicSystemInfo{
		LocalAddress:           localAddress(r),
		ServerName:             api.serverName(),
		Version:                jellyfinVersion,
		ProductName:            "Jellyfin Server",
		Id:                     api.serverID(r.Context()),
		StartupWizardCompleted: true,
	}
}

// localAddress reconstructs the base URL the client used (scheme/host from the request, honoring
// X-Forwarded-* headers, plus the mount path), advertised as LocalAddress. Jellify adopts it as
// its server base URL; without it its SDK api instance is undefined and sign-in crashes.
func localAddress(r *http.Request) string {
	scheme, host := server.ServerAddress(r)
	return scheme + "://" + host + path.Join(conf.Server.BasePath, consts.URLPathJellyfinAPI)
}

func (api *Router) getPublicSystemInfo(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, api.publicInfo(r))
}

func (api *Router) getSystemInfo(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, dto.SystemInfo{
		PublicSystemInfo:       api.publicInfo(r),
		SupportsLibraryMonitor: true,
	})
}

// ping answers /System/Ping with a bare plain-text server name (not JSON-quoted): Jellyfin's
// server does this and clients parse the raw body.
func (api *Router) ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(api.serverName()))
}

func (api *Router) quickConnectEnabled(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, false)
}
