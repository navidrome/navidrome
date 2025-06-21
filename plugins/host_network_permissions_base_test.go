package plugins

import (
	"net"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("networkPermissionsBase", func() {
	Describe("urlMatcher", func() {
		var matcher *urlMatcher

		BeforeEach(func() {
			matcher = newURLMatcher()
		})

		Describe("MatchesURLPattern", func() {
			DescribeTable("exact URL matching",
				func(requestURL, pattern string, expected bool) {
					result := matcher.MatchesURLPattern(requestURL, pattern)
					Expect(result).To(Equal(expected))
				},
				Entry("exact match", "https://api.example.com", "https://api.example.com", true),
				Entry("different domain", "https://api.example.com", "https://api.other.com", false),
				Entry("different scheme", "http://api.example.com", "https://api.example.com", false),
				Entry("different path", "https://api.example.com/v1", "https://api.example.com/v2", false),
			)

			DescribeTable("wildcard pattern matching",
				func(requestURL, pattern string, expected bool) {
					result := matcher.MatchesURLPattern(requestURL, pattern)
					Expect(result).To(Equal(expected))
				},
				Entry("universal wildcard", "https://api.example.com", "*", true),
				Entry("subdomain wildcard match", "https://api.example.com", "https://*.example.com", true),
				Entry("subdomain wildcard non-match", "https://api.other.com", "https://*.example.com", false),
				Entry("path wildcard match", "https://api.example.com/v1/users", "https://api.example.com/*", true),
				Entry("path wildcard non-match", "https://other.example.com/v1", "https://api.example.com/*", false),
				Entry("port wildcard match", "https://api.example.com:8080", "https://api.example.com:*", true),
			)
		})
	})

	Describe("isPrivateIP", func() {
		DescribeTable("IPv4 private IP detection",
			func(ip string, expected bool) {
				parsedIP := net.ParseIP(ip)
				Expect(parsedIP).ToNot(BeNil(), "Failed to parse IP: %s", ip)
				result := isPrivateIP(parsedIP)
				Expect(result).To(Equal(expected))
			},
			// Private IPv4 ranges
			Entry("10.0.0.1 (10.0.0.0/8)", "10.0.0.1", true),
			Entry("10.255.255.255 (10.0.0.0/8)", "10.255.255.255", true),
			Entry("172.16.0.1 (172.16.0.0/12)", "172.16.0.1", true),
			Entry("172.31.255.255 (172.16.0.0/12)", "172.31.255.255", true),
			Entry("192.168.1.1 (192.168.0.0/16)", "192.168.1.1", true),
			Entry("192.168.255.255 (192.168.0.0/16)", "192.168.255.255", true),
			Entry("127.0.0.1 (localhost)", "127.0.0.1", true),
			Entry("127.255.255.255 (localhost)", "127.255.255.255", true),
			Entry("169.254.1.1 (link-local)", "169.254.1.1", true),
			Entry("169.254.255.255 (link-local)", "169.254.255.255", true),

			// Public IPv4 addresses
			Entry("8.8.8.8 (Google DNS)", "8.8.8.8", false),
			Entry("1.1.1.1 (Cloudflare DNS)", "1.1.1.1", false),
			Entry("208.67.222.222 (OpenDNS)", "208.67.222.222", false),
			Entry("172.15.255.255 (just outside 172.16.0.0/12)", "172.15.255.255", false),
			Entry("172.32.0.1 (just outside 172.16.0.0/12)", "172.32.0.1", false),
		)

		DescribeTable("IPv6 private IP detection",
			func(ip string, expected bool) {
				parsedIP := net.ParseIP(ip)
				Expect(parsedIP).ToNot(BeNil(), "Failed to parse IP: %s", ip)
				result := isPrivateIP(parsedIP)
				Expect(result).To(Equal(expected))
			},
			// Private IPv6 ranges
			Entry("::1 (IPv6 localhost)", "::1", true),
			Entry("fe80::1 (link-local)", "fe80::1", true),
			Entry("fc00::1 (unique local)", "fc00::1", true),
			Entry("fd00::1 (unique local)", "fd00::1", true),

			// Public IPv6 addresses
			Entry("2001:4860:4860::8888 (Google DNS)", "2001:4860:4860::8888", false),
			Entry("2606:4700:4700::1111 (Cloudflare DNS)", "2606:4700:4700::1111", false),
		)
	})

	Describe("checkLocalNetwork", func() {
		DescribeTable("local network detection",
			func(urlStr string, shouldError bool, expectedErrorSubstring string) {
				parsedURL, err := url.Parse(urlStr)
				Expect(err).ToNot(HaveOccurred())

				err = checkLocalNetwork(parsedURL)
				if shouldError {
					Expect(err).To(HaveOccurred())
					if expectedErrorSubstring != "" {
						Expect(err.Error()).To(ContainSubstring(expectedErrorSubstring))
					}
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("localhost", "http://localhost:8080", true, "localhost"),
			Entry("127.0.0.1", "http://127.0.0.1:3000", true, "localhost"),
			Entry("::1", "http://[::1]:8080", true, "localhost"),
			Entry("private IP 192.168.1.100", "http://192.168.1.100", true, "private IP"),
			Entry("private IP 10.0.0.1", "http://10.0.0.1", true, "private IP"),
			Entry("private IP 172.16.0.1", "http://172.16.0.1", true, "private IP"),
			Entry("public IP 8.8.8.8", "http://8.8.8.8", false, ""),
			Entry("public domain", "https://api.example.com", false, ""),
		)
	})
})
