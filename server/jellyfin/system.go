package jellyfin

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"path"
	"strings"

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
		api.serverIDVal = newServerID()
		return api.serverIDVal
	}
	id, err := api.ds.Property(ctx).Get(consts.JellyfinServerIDKey)
	switch {
	case errors.Is(err, model.ErrNotFound):
		id = newServerID()
		if err := api.ds.Property(ctx).Put(consts.JellyfinServerIDKey, id); err != nil {
			log.Error(ctx, "Jellyfin API: could not persist server id", err)
			return id
		}
	case err != nil:
		log.Error(ctx, "Jellyfin API: could not read server id", err)
		return newServerID()
	}
	// Ids persisted before this change are dashed; normalize on read rather than rewriting the DB.
	api.serverIDVal = strings.ReplaceAll(id, "-", "")
	return api.serverIDVal
}

// newServerID returns a UUID in Jellyfin's no-dash GUID form (Guid.ToString("N")).
func newServerID() string {
	u := uuid.New()
	return hex.EncodeToString(u[:])
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

// getEndpointInfo answers /System/Endpoint, which Finamp's connection test uses to pick between a
// dual-connection setup's addresses; a missing IsInNetwork reads to it as "not a Jellyfin server".
func (api *Router) getEndpointInfo(w http.ResponseWriter, r *http.Request) {
	remote := remoteIP(r)
	api.ok(w, r, dto.EndPointInfo{
		IsLocal:     isSameMachine(r, remote),
		IsInNetwork: isInLocalNetwork(remote),
	})
}

// isInLocalNetwork mirrors Jellyfin's default LAN set (NetworkManager.UpdateSettings with no
// LocalNetworkSubnets configured): loopback, the RFC 1918 ranges, fc00::/7 and fe80::/10.
func isInLocalNetwork(ip netip.Addr) bool {
	return ip.IsLoopback() || ip.IsPrivate() || (ip.Is6() && ip.IsLinkLocalUnicast())
}

// isSameMachine mirrors Jellyfin's HttpContext.IsLocal(): the caller shares the connection's local
// address. The local address is missing in tests and unreliable behind a proxy, so fall back to loopback.
func isSameMachine(r *http.Request, remote netip.Addr) bool {
	local, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return remote.IsLoopback()
	}
	return parseIP(local.String()) == remote
}

// remoteIP parses RemoteAddr, which the RealIP middleware may have rewritten to a bare IP.
func remoteIP(r *http.Request) netip.Addr {
	return parseIP(r.RemoteAddr)
}

func parseIP(addr string) netip.Addr {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		addr = h
	}
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return netip.Addr{}
	}
	return ip.Unmap()
}

func (api *Router) quickConnectEnabled(w http.ResponseWriter, r *http.Request) {
	api.ok(w, r, false)
}
