package plugins

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HttpPermissions", func() {
	Describe("ParseHttpPermissions", func() {
		It("should require allowedUrls field", func() {
			permData := map[string]any{
				"reason": "To fetch data from APIs",
			}

			_, err := ParseHttpPermissions(permData)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("allowedUrls field is required"))
		})

		It("should parse comprehensive HTTP permissions", func() {
			permData := map[string]any{
				"reason": "To fetch data from APIs",
				"allowedUrls": map[string]any{
					"https://api.example.com": []any{"GET", "POST"},
					"https://*.last.fm":       []any{"GET"},
					"*":                       []any{"GET"},
				},
				"allowLocalNetwork": true,
			}

			perms, err := ParseHttpPermissions(permData)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms.Reason).To(Equal("To fetch data from APIs"))
			Expect(perms.AllowLocalNetwork).To(BeTrue())
			Expect(perms.AllowedUrls).To(HaveLen(3))
			Expect(perms.AllowedUrls["https://api.example.com"]).To(Equal([]string{"GET", "POST"}))
			Expect(perms.AllowedUrls["https://*.last.fm"]).To(Equal([]string{"GET"}))
			Expect(perms.AllowedUrls["*"]).To(Equal([]string{"GET"}))
		})

		DescribeTable("validation errors",
			func(permData map[string]any, expectedError string) {
				_, err := ParseHttpPermissions(permData)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedError))
			},
			Entry("missing reason field", map[string]any{
				"allowedUrls": map[string][]string{
					"https://api.example.com": {"GET"},
				},
			}, "reason is required"),

			Entry("empty reason field", map[string]any{
				"reason": "",
				"allowedUrls": map[string][]string{
					"*": {"*"},
				},
			}, "reason is required"),

			Entry("empty allowedUrls object", map[string]any{
				"reason":      "To fetch data from APIs",
				"allowedUrls": map[string]any{},
			}, "allowedUrls must contain at least one URL pattern"),

			Entry("invalid allowedUrls structure", map[string]any{
				"reason":      "To fetch data from APIs",
				"allowedUrls": "invalid",
			}, "allowedUrls must be a map"),

			Entry("invalid methods array", map[string]any{
				"reason": "To fetch data from APIs",
				"allowedUrls": map[string]any{
					"https://api.example.com": "invalid",
				},
			}, "methods for URL pattern"),
		)

		It("should normalize HTTP methods to uppercase", func() {
			permData := map[string]any{
				"reason": "To fetch data from APIs",
				"allowedUrls": map[string]any{
					"https://api.example.com": []any{"get", "post", "PUT"},
				},
			}

			perms, err := ParseHttpPermissions(permData)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms.AllowedUrls["https://api.example.com"]).To(Equal([]string{"GET", "POST", "PUT"}))
		})
	})

	Describe("IsRequestAllowed", func() {
		var perms *HttpPermissions

		Context("when no allowedUrls configured", func() {
			BeforeEach(func() {
				perms = &HttpPermissions{
					Reason:            "To fetch data from APIs",
					AllowLocalNetwork: true,
				}
			})

			It("should reject all requests", func() {
				err := perms.IsRequestAllowed("https://example.com", "GET")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no allowed URLs configured for plugin"))

				err = perms.IsRequestAllowed("http://localhost:8080", "POST")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no allowed URLs configured for plugin"))
			})
		})

		Context("with specific URL patterns", func() {
			BeforeEach(func() {
				perms = &HttpPermissions{
					Reason: "To fetch data from APIs",
					AllowedUrls: map[string][]string{
						"https://api.last.fm":           {"GET", "POST"},
						"https://*.last.fm":             {"GET"},
						"https://ws.audioscrobbler.com": {"*"},
						"https://api.*.com":             {"GET"},
						"https://example.com/api/*":     {"POST"},
					},
					AllowLocalNetwork: true,
				}
			})

			DescribeTable("specific pattern matching",
				func(url, method string, shouldSucceed bool, expectedError string) {
					err := perms.IsRequestAllowed(url, method)
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
						if expectedError != "" {
							Expect(err.Error()).To(ContainSubstring(expectedError))
						}
					}
				},
				// Exact URL matches
				Entry("exact URL with allowed method", "https://api.last.fm", "GET", true, ""),
				Entry("exact URL with another allowed method", "https://api.last.fm", "POST", true, ""),
				Entry("exact URL with disallowed method", "https://api.last.fm", "DELETE", false, "HTTP method DELETE not allowed"),

				// Subdomain wildcards
				Entry("subdomain wildcard match", "https://sub.last.fm", "GET", true, ""),
				Entry("subdomain wildcard with path", "https://another.last.fm/path", "GET", true, ""),
				Entry("subdomain wildcard wrong method", "https://sub.last.fm", "POST", false, "HTTP method POST not allowed"),

				// Domain wildcards
				Entry("domain wildcard match", "https://api.github.com", "GET", true, ""),
				Entry("domain wildcard with path", "https://api.stripe.com/charges", "GET", true, ""),

				// Path wildcards
				Entry("path wildcard match", "https://example.com/api/users", "POST", true, ""),
				Entry("path wildcard deep path", "https://example.com/api/orders/123", "POST", true, ""),

				// Wildcard method permissions
				Entry("wildcard methods GET", "https://ws.audioscrobbler.com", "GET", true, ""),
				Entry("wildcard methods POST", "https://ws.audioscrobbler.com", "POST", true, ""),
				Entry("wildcard methods DELETE", "https://ws.audioscrobbler.com", "DELETE", true, ""),

				// URLs that don't match any pattern
				Entry("no pattern match", "https://evil.com", "GET", false, "does not match any allowed URL patterns"),
				Entry("different domain no match", "https://random-site.org", "POST", false, "does not match any allowed URL patterns"),
			)
		})

		Context("with global wildcard", func() {
			BeforeEach(func() {
				perms = &HttpPermissions{
					Reason: "To test global wildcard behavior",
					AllowedUrls: map[string][]string{
						"https://api.last.fm": {"POST"}, // Specific pattern with limited methods
						"*":                   {"GET"},  // Global wildcard with limited methods
					},
					AllowLocalNetwork: true,
				}
			})

			DescribeTable("global wildcard behavior",
				func(url, method string, shouldSucceed bool, expectedError string) {
					err := perms.IsRequestAllowed(url, method)
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
						if expectedError != "" {
							Expect(err.Error()).To(ContainSubstring(expectedError))
						}
					}
				},
				// Specific pattern should take precedence over global wildcard
				Entry("specific pattern overrides global", "https://api.last.fm", "POST", true, ""),
				Entry("specific pattern rejects global wildcard method", "https://api.last.fm", "GET", false, "HTTP method GET not allowed"),

				// Global wildcard should catch everything else
				Entry("global wildcard allowed method", "https://random-site.com", "GET", true, ""),
				Entry("global wildcard disallowed method", "https://another-site.org", "POST", false, "HTTP method POST not allowed for URL pattern *"),
				Entry("global wildcard any domain", "https://totally-unknown.xyz", "GET", true, ""),
			)
		})

		Context("with local network restrictions", func() {
			BeforeEach(func() {
				perms = &HttpPermissions{
					Reason:            "To fetch data from APIs",
					AllowLocalNetwork: false,
					AllowedUrls: map[string][]string{
						"*": {"*"}, // Allow all URLs to test network restrictions
					},
				}
			})

			DescribeTable("network access control",
				func(url string, shouldSucceed bool, expectedError string) {
					err := perms.IsRequestAllowed(url, "GET")
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
						if expectedError != "" {
							Expect(err.Error()).To(ContainSubstring(expectedError))
						}
					}
				},
				// Localhost variants (should be blocked)
				Entry("localhost hostname", "http://localhost:8080", false, "requests to localhost are not allowed"),
				Entry("IPv4 localhost", "http://127.0.0.1:8080", false, "requests to localhost are not allowed"),
				Entry("IPv6 localhost", "http://[::1]:8080", false, "requests to localhost are not allowed"),

				// Private IP ranges (should be blocked)
				Entry("192.168.x.x range", "http://192.168.1.1", false, "requests to private IP addresses are not allowed"),
				Entry("10.x.x.x range", "http://10.0.0.1", false, "requests to private IP addresses are not allowed"),
				Entry("172.16.x.x range", "http://172.16.0.1", false, "requests to private IP addresses are not allowed"),

				// Public IPs (should be allowed)
				Entry("Google DNS", "https://8.8.8.8", true, ""),
				Entry("Cloudflare DNS", "https://1.1.1.1", true, ""),

				// Domain names (should be allowed)
				Entry("public domain", "https://example.com", true, ""),
				Entry("API domain", "https://api.github.com", true, ""),
			)

			// Test IP wildcard patterns specifically
			Context("with IP-specific patterns", func() {
				BeforeEach(func() {
					perms = &HttpPermissions{
						Reason:            "To test IP patterns",
						AllowLocalNetwork: true,
						AllowedUrls: map[string][]string{
							"http://192.168.*": {"GET", "POST"}, // Only specific IP pattern
						},
					}
				})

				DescribeTable("IP wildcard patterns",
					func(url, method string, shouldSucceed bool, expectedError string) {
						err := perms.IsRequestAllowed(url, method)
						if shouldSucceed {
							Expect(err).ToNot(HaveOccurred())
						} else {
							Expect(err).To(HaveOccurred())
							if expectedError != "" {
								Expect(err.Error()).To(ContainSubstring(expectedError))
							}
						}
					},
					Entry("192.168.10.1 GET allowed", "http://192.168.10.1", "GET", true, ""),
					Entry("192.168.1.100 POST allowed", "http://192.168.1.100", "POST", true, ""),
					Entry("192.168.10.1 DELETE rejected", "http://192.168.10.1", "DELETE", false, "HTTP method DELETE not allowed"),
					Entry("192.200.10.1 no match", "http://192.200.10.1", "GET", false, "does not match any allowed URL patterns"),
				)
			})
		})
	})

	Describe("Private IP Detection", func() {
		var perms *HttpPermissions

		BeforeEach(func() {
			perms = &HttpPermissions{}
		})

		DescribeTable("IP classification",
			func(ip string, expected bool) {
				result := perms.isPrivateIP(net.ParseIP(ip))
				Expect(result).To(Equal(expected))
			},
			// Private IPv4 ranges
			Entry("10.0.0.1", "10.0.0.1", true),
			Entry("172.16.0.1", "172.16.0.1", true),
			Entry("192.168.1.1", "192.168.1.1", true),
			Entry("127.0.0.1 (localhost)", "127.0.0.1", true),
			Entry("169.254.1.1 (link-local)", "169.254.1.1", true),

			// Public IPv4 addresses
			Entry("8.8.8.8 (Google DNS)", "8.8.8.8", false),
			Entry("1.1.1.1 (Cloudflare DNS)", "1.1.1.1", false),
			Entry("208.67.222.222 (OpenDNS)", "208.67.222.222", false),

			// Private IPv6 ranges
			Entry("::1 (IPv6 localhost)", "::1", true),
			Entry("fe80::1 (link-local)", "fe80::1", true),
			Entry("fc00::1 (unique local)", "fc00::1", true),

			// Public IPv6 addresses
			Entry("2001:4860:4860::8888 (Google DNS)", "2001:4860:4860::8888", false),
		)
	})

})
