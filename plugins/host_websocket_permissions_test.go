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
		It("should allow connections to explicitly allowed URLs", func() {
			perms := &WebSocketPermissions{
				NetworkPermissionsBase: &NetworkPermissionsBase{
					Reason:            "Test",
					AllowLocalNetwork: false,
				},
				AllowedUrls: []string{"wss://gateway.discord.gg"},
				matcher:     NewURLMatcher(),
			}

			err := perms.IsConnectionAllowed("wss://gateway.discord.gg")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject connections to disallowed URLs", func() {
			perms := &WebSocketPermissions{
				NetworkPermissionsBase: &NetworkPermissionsBase{
					Reason:            "Test",
					AllowLocalNetwork: false,
				},
				AllowedUrls: []string{"wss://allowed.com"},
				matcher:     NewURLMatcher(),
			}

			err := perms.IsConnectionAllowed("wss://disallowed.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not match any allowed URL patterns"))
		})

		It("should allow connections with wildcard patterns", func() {
			perms := &WebSocketPermissions{
				NetworkPermissionsBase: &NetworkPermissionsBase{
					Reason:            "Test",
					AllowLocalNetwork: false,
				},
				AllowedUrls: []string{"wss://*.example.com"},
				matcher:     NewURLMatcher(),
			}

			err := perms.IsConnectionAllowed("wss://sub.example.com")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject connections to local network when disabled", func() {
			perms := &WebSocketPermissions{
				NetworkPermissionsBase: &NetworkPermissionsBase{
					Reason:            "Test",
					AllowLocalNetwork: false,
				},
				AllowedUrls: []string{"*"},
				matcher:     NewURLMatcher(),
			}

			err := perms.IsConnectionAllowed("ws://localhost:8080")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("requests to localhost are not allowed"))
		})

		It("should allow connections to local network when enabled", func() {
			perms := &WebSocketPermissions{
				NetworkPermissionsBase: &NetworkPermissionsBase{
					Reason:            "Test",
					AllowLocalNetwork: true,
				},
				AllowedUrls: []string{"*"},
				matcher:     NewURLMatcher(),
			}

			err := perms.IsConnectionAllowed("ws://localhost:8080")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
