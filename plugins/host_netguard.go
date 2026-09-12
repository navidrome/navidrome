package plugins

import (
	"fmt"
	"net"
	"slices"
)

// checkPrivateDial runs at dial time on the resolved IP, so hostnames can't reach private addresses unless a
// literal IP/CIDR entry or a bare "*" (plugins targeting user-configured LAN services) allows it.
func checkPrivateDial(requiredHosts []string, address string) error {
	if slices.Contains(requiredHosts, "*") {
		return nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !isPrivateIP(ip) {
		return nil
	}
	for _, entry := range requiredHosts {
		if ipMatchesEntry(entry, ip) {
			return nil
		}
	}
	return fmt.Errorf("dial to private/loopback address %q blocked: requires an explicit IP or CIDR in requiredHosts", address)
}

func isHostInAllowlist(requiredHosts []string, hostname string) bool {
	ip := net.ParseIP(hostname)
	for _, pattern := range requiredHosts {
		if matchHostPattern(pattern, hostname) {
			return true
		}
		if ip != nil && ipMatchesEntry(pattern, ip) {
			return true
		}
	}
	return false
}

// ipMatchesEntry reports whether a requiredHosts entry is a literal IP or CIDR
// that covers ip. Hostname and wildcard entries never match.
func ipMatchesEntry(entry string, ip net.IP) bool {
	if _, cidr, err := net.ParseCIDR(entry); err == nil {
		return cidr.Contains(ip)
	}
	if entryIP := net.ParseIP(entry); entryIP != nil {
		return entryIP.Equal(ip)
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
