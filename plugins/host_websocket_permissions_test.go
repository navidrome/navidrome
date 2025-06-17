package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocketPermissions", func() {
	Describe("ParseWebSocketPermissions", func() {
		It("should parse valid WebSocket permissions with array format", func() {
			permData := map[string]any{
				"reason": "To connect to real-time services",
				"allowedUrls": []any{
					"wss://api.example.com",
					"ws://localhost:8080",
					"wss://*.example.com",
				},
				"allowLocalNetwork": true,
			}

			perms, err := ParseWebSocketPermissions(permData)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms.Reason).To(Equal("To connect to real-time services"))
			Expect(perms.AllowedUrls).To(HaveLen(3))
			Expect(perms.AllowedUrls).To(ContainElement("wss://api.example.com"))
			Expect(perms.AllowedUrls).To(ContainElement("ws://localhost:8080"))
			Expect(perms.AllowedUrls).To(ContainElement("wss://*.example.com"))
			Expect(perms.AllowLocalNetwork).To(BeTrue())
		})

		DescribeTable("parsing validation",
			func(permData map[string]any, shouldSucceed bool, expectedError string) {
				_, err := ParseWebSocketPermissions(permData)
				if shouldSucceed {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(HaveOccurred())
					if expectedError != "" {
						Expect(err.Error()).To(ContainSubstring(expectedError))
					}
				}
			},
			Entry("missing allowedUrls", map[string]any{
				"reason": "Test reason",
			}, false, "allowedUrls field is required"),
			Entry("missing reason", map[string]any{
				"allowedUrls": []any{"wss://example.com"},
			}, false, "reason is required"),
			Entry("empty allowedUrls array", map[string]any{
				"reason":      "Test reason",
				"allowedUrls": []any{},
			}, false, "allowedUrls must contain at least one URL pattern"),
			Entry("invalid allowedUrls type", map[string]any{
				"reason":      "Test reason",
				"allowedUrls": "invalid",
			}, false, "allowedUrls must be an array"),
			Entry("invalid URL in array", map[string]any{
				"reason":      "Test reason",
				"allowedUrls": []any{123}, // non-string
			}, false, "URL pattern at index 0 must be a string"),
		)

		It("should handle allowLocalNetwork defaults", func() {
			permData := map[string]any{
				"reason":      "Test reason",
				"allowedUrls": []any{"wss://example.com"},
				// allowLocalNetwork not specified - should default to false
			}

			perms, err := ParseWebSocketPermissions(permData)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms.AllowLocalNetwork).To(BeFalse()) // Default value
		})
	})

	Describe("IsConnectionAllowed", func() {
		var perms *WebSocketPermissions

		Context("with exact URL matches", func() {
			BeforeEach(func() {
				perms = &WebSocketPermissions{
					Reason: "Test permissions",
					AllowedUrls: []string{
						"wss://api.example.com",
						"ws://localhost:8080",
						"wss://secure.example.com:443",
					},
					AllowLocalNetwork: true, // Allow local network for this test
					matcher:           NewURLMatcher(),
				}
			})

			DescribeTable("exact URL matching",
				func(url string, shouldSucceed bool) {
					err := perms.IsConnectionAllowed(url)
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
						// Don't check specific error message since it could be pattern mismatch or localhost restriction
					}
				},
				Entry("exact wss match", "wss://api.example.com", true),
				Entry("exact ws match", "ws://localhost:8080", true),
				Entry("exact wss with port", "wss://secure.example.com:443", true),
				Entry("non-matching domain", "wss://malicious.com", false),
				Entry("wrong scheme", "https://api.example.com", false),
				Entry("wrong port", "ws://localhost:3000", false),
			)
		})

		Context("with wildcard patterns", func() {
			BeforeEach(func() {
				perms = &WebSocketPermissions{
					Reason: "Test permissions",
					AllowedUrls: []string{
						"wss://*.example.com",
						"ws://localhost:*",
						"wss://api.*.com",
						"*://*.dev.local", // Any scheme
					},
					AllowLocalNetwork: true,
					matcher:           NewURLMatcher(),
				}
			})

			DescribeTable("wildcard pattern matching",
				func(url string, shouldSucceed bool) {
					err := perms.IsConnectionAllowed(url)
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("does not match any allowed URL patterns"))
					}
				},
				// Subdomain wildcards
				Entry("subdomain wildcard match", "wss://api.example.com", true),
				Entry("subdomain wildcard match 2", "wss://cdn.example.com", true),
				Entry("subdomain wildcard non-match", "wss://other.different.org", false),

				// Port wildcards
				Entry("port wildcard match", "ws://localhost:8080", true),
				Entry("port wildcard match 2", "ws://localhost:3000", true),
				Entry("port wildcard non-match", "ws://127.0.0.1:8080", false),

				// Multiple wildcards
				Entry("multiple wildcard match", "wss://api.staging.com", true),
				Entry("multiple wildcard match 2", "wss://api.prod.com", true),
				Entry("multiple wildcard non-match", "wss://api.staging.net", false),

				// Scheme wildcards
				Entry("any scheme match ws", "ws://service.dev.local", true),
				Entry("any scheme match wss", "wss://api.dev.local", true),
				Entry("any scheme non-match", "wss://api.dev.remote", false),
			)
		})

		Context("with universal wildcard", func() {
			BeforeEach(func() {
				perms = &WebSocketPermissions{
					Reason: "Test permissions",
					AllowedUrls: []string{
						"*", // Allow everything
					},
					AllowLocalNetwork: true,
					matcher:           NewURLMatcher(),
				}
			})

			It("should allow any WebSocket URL", func() {
				urls := []string{
					"wss://api.example.com",
					"ws://localhost:8080",
					"wss://any.domain.com:443",
					"ws://192.168.1.100:3000",
				}

				for _, url := range urls {
					err := perms.IsConnectionAllowed(url)
					Expect(err).ToNot(HaveOccurred(), "URL should be allowed: %s", url)
				}
			})
		})

		Context("with local network restrictions", func() {
			BeforeEach(func() {
				perms = &WebSocketPermissions{
					Reason: "Test permissions",
					AllowedUrls: []string{
						"*", // Allow all URLs so local network check is triggered
					},
					AllowLocalNetwork: false,
					matcher:           NewURLMatcher(),
				}
			})

			DescribeTable("local network access control",
				func(url string, shouldSucceed bool, expectedErrorSubstring string) {
					err := perms.IsConnectionAllowed(url)
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
						if expectedErrorSubstring != "" {
							Expect(err.Error()).To(ContainSubstring(expectedErrorSubstring))
						}
					}
				},
				// Should be blocked
				Entry("localhost", "ws://localhost:8080", false, "localhost"),
				Entry("127.0.0.1", "ws://127.0.0.1:3000", false, "localhost"),
				Entry("IPv6 localhost", "ws://[::1]:8080", false, "localhost"),
				Entry("private IP 192.168", "ws://192.168.1.100:8080", false, "private IP"),
				Entry("private IP 10.x", "ws://10.0.0.1:8080", false, "private IP"),
				Entry("private IP 172.16", "ws://172.16.0.1:8080", false, "private IP"),

				// Should be allowed
				Entry("public domain", "wss://api.example.com", true, ""),
				Entry("public IP", "ws://8.8.8.8:8080", true, ""),
			)
		})

		Context("with mixed patterns and local network control", func() {
			BeforeEach(func() {
				perms = &WebSocketPermissions{
					Reason: "Test permissions",
					AllowedUrls: []string{
						"wss://api.example.com",
						"ws://localhost:*", // Explicitly allow localhost on any port
						"wss://*.public.com",
					},
					AllowLocalNetwork: false, // But generally restrict local network
					matcher:           NewURLMatcher(),
				}
			})

			It("should block local URLs when allowLocalNetwork is false, even if explicitly configured", func() {
				// This tests that local network restrictions take precedence over URL patterns
				err := perms.IsConnectionAllowed("ws://localhost:8080")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("localhost"))

				err = perms.IsConnectionAllowed("ws://localhost:3000")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("localhost"))
			})

			It("should block non-explicitly-allowed local URLs", func() {
				// This should be blocked because it's localhost and allowLocalNetwork is false
				err := perms.IsConnectionAllowed("ws://127.0.0.1:8080")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("localhost"))
			})

			It("should allow public URLs matching patterns", func() {
				err := perms.IsConnectionAllowed("wss://api.example.com")
				Expect(err).ToNot(HaveOccurred())

				err = perms.IsConnectionAllowed("wss://cdn.public.com")
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
