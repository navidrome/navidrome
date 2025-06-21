package plugins

import (
	"github.com/navidrome/navidrome/plugins/schema"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP Permissions", func() {
	Describe("parseHTTPPermissions", func() {
		It("should parse valid HTTP permissions", func() {
			permData := &schema.PluginManifestPermissionsHttp{
				Reason:            "Need to fetch album artwork",
				AllowLocalNetwork: false,
				AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
					"https://api.example.com/*": {
						schema.PluginManifestPermissionsHttpAllowedUrlsValueElemGET,
						schema.PluginManifestPermissionsHttpAllowedUrlsValueElemPOST,
					},
					"https://cdn.example.com/*": {
						schema.PluginManifestPermissionsHttpAllowedUrlsValueElemGET,
					},
				},
			}

			perms, err := parseHTTPPermissions(permData)
			Expect(err).To(BeNil())
			Expect(perms).ToNot(BeNil())
			Expect(perms.AllowLocalNetwork).To(BeFalse())
			Expect(perms.AllowedUrls).To(HaveLen(2))
			Expect(perms.AllowedUrls["https://api.example.com/*"]).To(Equal([]string{"GET", "POST"}))
			Expect(perms.AllowedUrls["https://cdn.example.com/*"]).To(Equal([]string{"GET"}))
		})

		It("should fail if allowedUrls is empty", func() {
			permData := &schema.PluginManifestPermissionsHttp{
				Reason:            "Need to fetch album artwork",
				AllowLocalNetwork: false,
				AllowedUrls:       map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{},
			}

			_, err := parseHTTPPermissions(permData)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("allowedUrls must contain at least one URL pattern"))
		})

		It("should handle method enum types correctly", func() {
			permData := &schema.PluginManifestPermissionsHttp{
				Reason:            "Need to fetch album artwork",
				AllowLocalNetwork: false,
				AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
					"https://api.example.com/*": {
						schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard, // "*"
					},
				},
			}

			perms, err := parseHTTPPermissions(permData)
			Expect(err).To(BeNil())
			Expect(perms.AllowedUrls["https://api.example.com/*"]).To(Equal([]string{"*"}))
		})
	})

	Describe("IsRequestAllowed", func() {
		var perms *httpPermissions

		Context("HTTP method-specific validation", func() {
			BeforeEach(func() {
				perms = &httpPermissions{
					networkPermissionsBase: &networkPermissionsBase{
						Reason:            "Test permissions",
						AllowLocalNetwork: false,
					},
					AllowedUrls: map[string][]string{
						"https://api.example.com":     {"GET", "POST"},
						"https://upload.example.com":  {"PUT", "PATCH"},
						"https://admin.example.com":   {"DELETE"},
						"https://webhook.example.com": {"*"},
					},
					matcher: newURLMatcher(),
				}
			})

			DescribeTable("method-specific access control",
				func(url, method string, shouldSucceed bool) {
					err := perms.IsRequestAllowed(url, method)
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
					}
				},
				// Allowed methods
				Entry("GET to api", "https://api.example.com", "GET", true),
				Entry("POST to api", "https://api.example.com", "POST", true),
				Entry("PUT to upload", "https://upload.example.com", "PUT", true),
				Entry("PATCH to upload", "https://upload.example.com", "PATCH", true),
				Entry("DELETE to admin", "https://admin.example.com", "DELETE", true),
				Entry("any method to webhook", "https://webhook.example.com", "OPTIONS", true),
				Entry("any method to webhook", "https://webhook.example.com", "HEAD", true),

				// Disallowed methods
				Entry("DELETE to api", "https://api.example.com", "DELETE", false),
				Entry("GET to upload", "https://upload.example.com", "GET", false),
				Entry("POST to admin", "https://admin.example.com", "POST", false),
			)
		})

		Context("case insensitive method handling", func() {
			BeforeEach(func() {
				perms = &httpPermissions{
					networkPermissionsBase: &networkPermissionsBase{
						Reason:            "Test permissions",
						AllowLocalNetwork: false,
					},
					AllowedUrls: map[string][]string{
						"https://api.example.com": {"GET", "POST"}, // Both uppercase for consistency
					},
					matcher: newURLMatcher(),
				}
			})

			DescribeTable("case insensitive method matching",
				func(method string, shouldSucceed bool) {
					err := perms.IsRequestAllowed("https://api.example.com", method)
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
					}
				},
				Entry("uppercase GET", "GET", true),
				Entry("lowercase get", "get", true),
				Entry("mixed case Get", "Get", true),
				Entry("uppercase POST", "POST", true),
				Entry("lowercase post", "post", true),
				Entry("mixed case Post", "Post", true),
				Entry("disallowed method", "DELETE", false),
			)
		})

		Context("with complex URL patterns and HTTP methods", func() {
			BeforeEach(func() {
				perms = &httpPermissions{
					networkPermissionsBase: &networkPermissionsBase{
						Reason:            "Test permissions",
						AllowLocalNetwork: false,
					},
					AllowedUrls: map[string][]string{
						"https://api.example.com/v1/*":     {"GET"},
						"https://api.example.com/v1/users": {"POST", "PUT"},
						"https://*.example.com/public/*":   {"GET", "HEAD"},
						"https://admin.*.example.com":      {"*"},
					},
					matcher: newURLMatcher(),
				}
			})

			DescribeTable("complex pattern and method combinations",
				func(url, method string, shouldSucceed bool) {
					err := perms.IsRequestAllowed(url, method)
					if shouldSucceed {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(HaveOccurred())
					}
				},
				// Path wildcards with specific methods
				Entry("GET to v1 path", "https://api.example.com/v1/posts", "GET", true),
				Entry("POST to v1 path", "https://api.example.com/v1/posts", "POST", false),
				Entry("POST to specific users endpoint", "https://api.example.com/v1/users", "POST", true),
				Entry("PUT to specific users endpoint", "https://api.example.com/v1/users", "PUT", true),
				Entry("DELETE to specific users endpoint", "https://api.example.com/v1/users", "DELETE", false),

				// Subdomain wildcards with specific methods
				Entry("GET to public path on subdomain", "https://cdn.example.com/public/assets", "GET", true),
				Entry("HEAD to public path on subdomain", "https://static.example.com/public/files", "HEAD", true),
				Entry("POST to public path on subdomain", "https://api.example.com/public/upload", "POST", false),

				// Admin subdomain with all methods
				Entry("GET to admin subdomain", "https://admin.prod.example.com", "GET", true),
				Entry("POST to admin subdomain", "https://admin.staging.example.com", "POST", true),
				Entry("DELETE to admin subdomain", "https://admin.dev.example.com", "DELETE", true),
			)
		})
	})
})
