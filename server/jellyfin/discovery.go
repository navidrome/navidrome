package jellyfin

import (
	"cmp"
	"context"
	"encoding/json"
	"net"
	"path"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/gg"
)

// Jellyfin clients broadcast this query text to this UDP port.
const (
	discoveryPort  = 7359
	discoveryQuery = "who is jellyfinserver?"
)

type discoveryInfo struct {
	Address         string  `json:"Address"`
	Id              string  `json:"Id"`
	Name            string  `json:"Name"`
	EndpointAddress *string `json:"EndpointAddress"`
}

// ServeDiscovery serves until ctx is done; it returns an error only when the port can't be bound.
func (api *Router) ServeDiscovery(ctx context.Context) error {
	// udp4 only: a dual-stack bind can share the port with another server and never get a packet.
	conn, err := net.ListenPacket("udp4", net.JoinHostPort("0.0.0.0", strconv.Itoa(discoveryPort)))
	if err != nil {
		return err
	}
	log.Info(ctx, "Jellyfin API: listening for auto-discovery broadcasts", "port", discoveryPort)
	api.ServeDiscoveryOn(ctx, conn)
	return nil
}

// ServeDiscoveryOn answers discovery queries on conn until ctx is done, then closes conn.
func (api *Router) ServeDiscoveryOn(ctx context.Context, conn net.PacketConn) {
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	buf := make([]byte, 1024)
	for {
		n, remote, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() == nil {
				log.Error(ctx, "Jellyfin API: auto-discovery listener stopped", err)
			}
			return
		}
		if !strings.Contains(strings.ToLower(string(buf[:n])), discoveryQuery) {
			continue
		}
		info := discoveryInfo{Address: api.discoveryAddress(remote), Id: api.serverID(ctx), Name: api.serverName()}
		res, _ := json.Marshal(info)
		log.Debug(ctx, "Jellyfin API: answering auto-discovery request", "from", remote.String(), "address", info.Address)
		if _, err := conn.WriteTo(res, remote); err != nil {
			log.Debug(ctx, "Jellyfin API: could not answer auto-discovery request", "to", remote.String(), err)
		}
	}
}

func (api *Router) discoveryAddress(remote net.Addr) string {
	scheme := cmp.Or(conf.Server.BaseScheme, gg.If(conf.Server.TLSEnabled(), "https", "http"))
	host := conf.Server.BaseHost
	if host == "" {
		host = net.JoinHostPort(localIPFor(remote), strconv.Itoa(conf.Server.Port))
	}
	return scheme + "://" + host + path.Join(conf.Server.BasePath, consts.URLPathJellyfinAPI)
}

// On a multi-homed host, only the interface that routes to the requester is reachable by it.
func localIPFor(remote net.Addr) string {
	if ip := net.ParseIP(conf.Server.Address); ip != nil && !ip.IsUnspecified() {
		return ip.String()
	}
	c, err := net.Dial("udp", remote.String())
	if err != nil {
		return conf.Server.Address
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String()
}
