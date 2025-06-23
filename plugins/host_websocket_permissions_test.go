package plugins

import (
	"github.com/navidrome/navidrome/plugins/schema"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocket Permissions", func() {
	Describe("parseWebSocketPermissions", func() {
		It("should parse valid WebSocket permissions", func() {
			permData := &schema.PluginManifestPermissionsWebsocket{
				Reason:            "Need to connect to WebSocket API",
				AllowLocalNetwork: false,
				AllowedUrls:       []string{"wss://api.example.com/ws", "wss://cdn.example.com/*"},
			}

			perms, err := parseWebSocketPermissions(permData)
			Expect(err).To(BeNil())
			Expect(perms).ToNot(BeNil())
			Expect(perms.AllowLocalNetwork).To(BeFalse())
			Expect(perms.AllowedUrls).To(Equal([]string{"wss://api.example.com/ws", "wss://cdn.example.com/*"}))
		})

		It("should fail if allowedUrls is empty", func() {
			permData := &schema.PluginManifestPermissionsWebsocket{
				Reason:            "Need to connect to WebSocket API",
				AllowLocalNetwork: false,
				AllowedUrls:       []string{},
			}

			_, err := parseWebSocketPermissions(permData)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("allowedUrls must contain at least one URL pattern"))
		})

		It("should handle wildcard patterns", func() {
			permData := &schema.PluginManifestPermissionsWebsocket{
				Reason:            "Need to connect to any WebSocket",
				AllowLocalNetwork: true,
				AllowedUrls:       []string{"wss://*"},
			}

			perms, err := parseWebSocketPermissions(permData)
			Expect(err).To(BeNil())
			Expect(perms.AllowLocalNetwork).To(BeTrue())
			Expect(perms.AllowedUrls).To(Equal([]string{"wss://*"}))
		})

		Context("URL matching", func() {
			var perms *webSocketPermissions

			BeforeEach(func() {
				permData := &schema.PluginManifestPermissionsWebsocket{
					Reason:            "Need to connect to external services",
					AllowLocalNetwork: true,
					AllowedUrls:       []string{"wss://api.example.com/*", "ws://localhost:8080"},
				}
				var err error
				perms, err = parseWebSocketPermissions(permData)
				Expect(err).To(BeNil())
			})

			It("should allow connections to URLs matching patterns", func() {
				err := perms.IsConnectionAllowed("wss://api.example.com/v1/stream")
				Expect(err).To(BeNil())

				err = perms.IsConnectionAllowed("ws://localhost:8080")
				Expect(err).To(BeNil())
			})

			It("should deny connections to URLs not matching patterns", func() {
				err := perms.IsConnectionAllowed("wss://malicious.com/stream")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not match any allowed URL patterns"))
			})
		})
	})
})
