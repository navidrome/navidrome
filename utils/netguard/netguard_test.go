package netguard_test

import (
	"net"
	"net/netip"

	"github.com/navidrome/navidrome/utils/netguard"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("netguard", func() {
	DescribeTable("IsPrivateIP",
		func(addr string, expected bool) {
			Expect(netguard.IsPrivateIP(net.ParseIP(addr))).To(Equal(expected))
		},
		Entry("IPv4 loopback", "127.0.0.1", true),
		Entry("IPv4 loopback range", "127.0.0.2", true),
		Entry("IPv6 loopback", "::1", true),
		Entry("IPv4-mapped IPv6 loopback", "::ffff:127.0.0.1", true),
		Entry("10.x", "10.1.2.3", true),
		Entry("172.16.x", "172.16.0.1", true),
		Entry("192.168.x", "192.168.1.10", true),
		Entry("IPv6 unique local", "fd00::1", true),
		Entry("link-local (cloud metadata)", "169.254.169.254", true),
		Entry("IPv6 link-local", "fe80::1", true),
		Entry("link-local multicast", "224.0.0.1", true),
		Entry("IPv4 unspecified", "0.0.0.0", true),
		Entry("IPv6 unspecified", "::", true),
		Entry("public IPv4", "93.184.216.34", false),
		Entry("172.32.x is outside the private block", "172.32.0.1", false),
		Entry("public IPv6", "2606:4700:4700::1111", false),
	)

	Describe("DialControl", func() {
		It("rejects private and loopback addresses", func() {
			control := netguard.DialControl()
			for _, addr := range []string{"127.0.0.1:80", "[::1]:443", "169.254.169.254:80", "10.0.0.1:8080", "0.0.0.0:80"} {
				Expect(control("tcp", addr, nil)).To(MatchError(netguard.ErrPrivateAddress), addr)
			}
		})

		It("allows public addresses", func() {
			control := netguard.DialControl()
			Expect(control("tcp", "93.184.216.34:443", nil)).To(Succeed())
			Expect(control("tcp6", "[2606:4700:4700::1111]:443", nil)).To(Succeed())
		})

		It("allows private addresses covered by an allowed prefix, and only those", func() {
			control := netguard.DialControl(netip.MustParsePrefix("127.0.0.1/32"))
			Expect(control("tcp", "127.0.0.1:8080", nil)).To(Succeed())
			Expect(control("tcp", "127.0.0.2:8080", nil)).To(MatchError(netguard.ErrPrivateAddress))
		})

		It("matches IPv4-mapped IPv6 addresses against IPv4 prefixes", func() {
			control := netguard.DialControl(netip.MustParsePrefix("127.0.0.0/8"))
			Expect(control("tcp", "[::ffff:127.0.0.1]:80", nil)).To(Succeed())
		})

		It("fails closed when the address is not an IP", func() {
			Expect(netguard.DialControl()("tcp", "localhost:80", nil)).To(HaveOccurred())
			Expect(netguard.DialControl()("tcp", "garbage", nil)).To(HaveOccurred())
		})
	})
})
