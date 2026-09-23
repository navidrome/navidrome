package jellyfin

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
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

// Discovery answers LAN auto-discovery broadcasts with the same identity the Router reports.
type Discovery struct {
	ds          model.DataStore
	serverIDVal string
}

func NewDiscovery(ds model.DataStore) *Discovery {
	return &Discovery{ds: ds}
}

func (d *Discovery) serverID(ctx context.Context) string {
	return resolveServerID(ctx, d.ds, &d.serverIDVal)
}

// Serve runs until ctx is done. A failed bind is only logged: discovery is best-effort.
func (d *Discovery) Serve(ctx context.Context) {
	if !hasAdvertisableAddress() {
		log.Warn(ctx, "Jellyfin API: auto-discovery is off, a unix socket server needs a BaseURL with a host to advertise")
		return
	}
	// udp4 only: a dual-stack bind can share the port with another server and never get a packet.
	conn, err := net.ListenPacket("udp4", net.JoinHostPort("0.0.0.0", strconv.Itoa(discoveryPort)))
	if err != nil {
		log.Warn(ctx, "Jellyfin API: auto-discovery is off, the UDP port is unavailable. Is another Jellyfin server running?", "port", discoveryPort, err)
		return
	}
	log.Info(ctx, "Jellyfin API: listening for auto-discovery broadcasts", "port", discoveryPort)
	d.ServeOn(ctx, conn)
}

// ServeOn answers discovery queries on conn until ctx is done, then closes conn.
func (d *Discovery) ServeOn(ctx context.Context, conn net.PacketConn) {
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
		info := discoveryInfo{Address: discoveryAddress(ctx, remote), Id: d.serverID(ctx), Name: serverName()}
		res, _ := json.Marshal(info)
		log.Debug(ctx, "Jellyfin API: answering auto-discovery request", "from", remote.String(), "address", info.Address)
		if _, err := conn.WriteTo(res, remote); err != nil {
			log.Debug(ctx, "Jellyfin API: could not answer auto-discovery request", "to", remote.String(), err)
		}
	}
}

// Behind a unix socket nothing listens on Port, so only a BaseURL host gives clients an address.
func hasAdvertisableAddress() bool {
	return conf.Server.BaseHost != "" || !strings.HasPrefix(conf.Server.Address, "unix:")
}

func discoveryAddress(ctx context.Context, remote net.Addr) string {
	scheme := gg.If(conf.Server.TLSEnabled(), "https", "http")
	host := net.JoinHostPort(localIPFor(remote), strconv.Itoa(conf.Server.Port))
	return publicurl.AbsoluteURL(request.WithServerAddress(ctx, scheme, host), consts.URLPathJellyfinAPI, nil)
}

// On a multi-homed host, only the interface that routes to the requester is reachable by it.
func localIPFor(remote net.Addr) string {
	if ip := parseIP(conf.Server.Address); ip.IsValid() && !ip.IsUnspecified() {
		return ip.String()
	}
	c, err := net.Dial("udp", remote.String())
	if err != nil {
		return conf.Server.Address
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String()
}
