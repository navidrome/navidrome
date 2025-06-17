package plugins

import (
	"net"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkPermissionsBase", func() {
	Describe("ParseNetworkPermissionsBase", func() {
		It("should require allowedUrls field", func() {
			permData := map[string]any{
				"reason": "Test reason",
			}

			_, err := ParseNetworkPermissionsBase(permData)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("allowedUrls field is required"))
		})

		It("should require reason field", func() {
			permData := map[string]any{
				"allowedUrls": map[string]any{
					"https://example.com": []any{"GET"},
				},
			}

			_, err := ParseNetworkPermissionsBase(permData)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("reason is required"))
		})

		It("should parse valid network permissions", func() {
			permData := map[string]any{
				"reason": "Test permissions",
				"allowedUrls": map[string]any{
					"https://api.example.com": []any{"GET", "POST"},
					"https://*.example.com":   []any{"*"},
				},
				"allowLocalNetwork": true,
			}

			perms, err := ParseNetworkPermissionsBase(permData)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms.Reason).To(Equal("Test permissions"))
			Expect(perms.AllowedUrls).To(HaveLen(2))
			Expect(perms.AllowedUrls["https://api.example.com"]).To(Equal([]string{"GET", "POST"}))
			Expect(perms.AllowedUrls["https://*.example.com"]).To(Equal([]string{"*"}))
			Expect(perms.AllowLocalNetwork).To(BeTrue())
		})

		DescribeTable("invalid allowedUrls formats",
			func(allowedUrls any, expectedError string) {
				permData := map[string]any{
					"reason":      "Test reason",
					"allowedUrls": allowedUrls,
				}

				_, err := ParseNetworkPermissionsBase(permData)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedError))
			},
			Entry("string instead of map", "invalid", "allowedUrls must be a map"),
			Entry("invalid operations array", map[string]any{
				"https://api.example.com": "invalid",
			}, "operations for URL pattern"),
		)
	})

	Describe("URLMatcher", func() {
		var matcher *URLMatcher

		BeforeEach(func() {
			matcher = NewURLMatcher()
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

	Describe("IsPrivateIP", func() {
		DescribeTable("IPv4 private IP detection",
			func(ip string, expected bool) {
				parsedIP := net.ParseIP(ip)
				Expect(parsedIP).ToNot(BeNil(), "Failed to parse IP: %s", ip)
				result := IsPrivateIP(parsedIP)
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
				result := IsPrivateIP(parsedIP)
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

	Describe("CheckLocalNetwork", func() {
		DescribeTable("local network detection",
			func(urlStr string, shouldError bool, expectedErrorSubstring string) {
				parsedURL, err := url.Parse(urlStr)
				Expect(err).ToNot(HaveOccurred())

				err = CheckLocalNetwork(parsedURL)
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

	Describe("IsRequestAllowed", func() {
		var base *NetworkPermissionsBase

		Context("with exact URL matches", func() {
			BeforeEach(func() {
				base = &NetworkPermissionsBase{
					Reason: "Test permissions",
					AllowedUrls: map[string][]string{
						"https://api.example.com": {"GET", "POST"},
						"ws://localhost:8080":     {"*"},
					},
					AllowLocalNetwork: true, // Allow local network for this test
				}
			})

			It("should allow requests to exact URL matches with correct operations", func() {
				err := base.IsRequestAllowed("https://api.example.com", "GET")
				Expect(err).ToNot(HaveOccurred())

				err = base.IsRequestAllowed("https://api.example.com", "POST")
				Expect(err).ToNot(HaveOccurred())

				err = base.IsRequestAllowed("ws://localhost:8080", "CONNECT")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should reject requests with disallowed operations", func() {
				err := base.IsRequestAllowed("https://api.example.com", "DELETE")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("operation DELETE not allowed"))
			})

			It("should reject requests to non-matching URLs", func() {
				err := base.IsRequestAllowed("https://malicious.com", "GET")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not match any allowed URL patterns"))
			})
		})

		Context("with wildcard patterns", func() {
			BeforeEach(func() {
				base = &NetworkPermissionsBase{
					Reason: "Test permissions",
					AllowedUrls: map[string][]string{
						"https://*.example.com": {"GET"},
						"*":                     {"*"},
					},
					AllowLocalNetwork: true,
				}
			})

			It("should allow requests matching wildcard patterns", func() {
				err := base.IsRequestAllowed("https://api.example.com", "GET")
				Expect(err).ToNot(HaveOccurred())

				err = base.IsRequestAllowed("https://cdn.example.com", "GET")
				Expect(err).ToNot(HaveOccurred())

				err = base.IsRequestAllowed("https://any.domain.com", "POST")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with local network restrictions", func() {
			BeforeEach(func() {
				base = &NetworkPermissionsBase{
					Reason: "Test permissions",
					AllowedUrls: map[string][]string{
						"*": {"*"}, // Allow all URLs so local network check is triggered
					},
					AllowLocalNetwork: false,
				}
			})

			It("should reject requests to localhost when allowLocalNetwork is false", func() {
				err := base.IsRequestAllowed("http://localhost:8080", "GET")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("localhost"))

				err = base.IsRequestAllowed("http://127.0.0.1:3000", "GET")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("localhost"))
			})

			It("should reject requests to private IP addresses when allowLocalNetwork is false", func() {
				err := base.IsRequestAllowed("http://192.168.1.100", "GET")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("private IP"))
			})
		})

		Context("with no allowed URLs", func() {
			BeforeEach(func() {
				base = &NetworkPermissionsBase{
					Reason:      "Test permissions",
					AllowedUrls: map[string][]string{}, // Empty but not nil
				}
			})

			It("should reject all requests when no URLs are configured", func() {
				err := base.IsRequestAllowed("https://example.com", "GET")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no allowed URLs"))
			})
		})
	})
})
