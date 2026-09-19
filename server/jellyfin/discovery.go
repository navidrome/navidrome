package jellyfin

import (
	"cmp"
	"net"
	"path"
	"strconv"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils/gg"
)

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
