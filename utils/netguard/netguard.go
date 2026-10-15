// Package netguard keeps outbound connections driven by untrusted input away from internal addresses.
package netguard

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"syscall"
)

var ErrPrivateAddress = errors.New("dial to private/loopback address blocked")

// IsPrivateIP reports whether ip is loopback, private, link-local or unspecified.
func IsPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// DialControl returns a net.Dialer Control hook that rejects private addresses not covered by allowed.
// It sees the resolved IP, so DNS names, redirects and rebinding cannot get around it.
func DialControl(allowed ...netip.Prefix) func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		ap, err := netip.ParseAddrPort(address)
		if err != nil {
			return fmt.Errorf("%w: unparseable address %q: %w", ErrPrivateAddress, address, err)
		}
		addr := ap.Addr().Unmap()
		if !IsPrivateIP(addr.AsSlice()) {
			return nil
		}
		for _, p := range allowed {
			if p.Contains(addr) {
				return nil
			}
		}
		return fmt.Errorf("%w: %s", ErrPrivateAddress, address)
	}
}
