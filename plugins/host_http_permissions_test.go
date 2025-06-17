package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTPPermissions", func() {
	Describe("ParseHTTPPermissions", func() {
		It("should parse valid HTTP permissions", func() {
			permData := map[string]any{
				"reason": "To fetch data from APIs",
				"allowedUrls": map[string]any{
					"https://api.example.com": []any{"GET", "POST"},
					"https://*.example.com":   []any{"*"},
				},
				"allowLocalNetwork": true,
			}

			perms, err := ParseHTTPPermissions(permData)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms.Reason).To(Equal("To fetch data from APIs"))
			Expect(perms.AllowedUrls).To(HaveLen(2))
			Expect(perms.AllowedUrls["https://api.example.com"]).To(Equal([]string{"GET", "POST"}))
			Expect(perms.AllowedUrls["https://*.example.com"]).To(Equal([]string{"*"}))
			Expect(perms.AllowLocalNetwork).To(BeTrue())
		})

		DescribeTable("HTTP method validation",
			func(methods []any, shouldSucceed bool, expectedError string) {
				permData := map[string]any{
					"reason": "Test permissions",
					"allowedUrls": map[string]any{
						"https://api.example.com": methods,
					},
				}

				_, err := ParseHTTPPermissions(permData)
				if shouldSucceed {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(HaveOccurred())
					if expectedError != "" {
						Expect(err.Error()).To(ContainSubstring(expectedError))
					}
				}
			},
			Entry("valid HTTP methods", []any{"GET", "POST", "PUT", "DELETE"}, true, ""),
			Entry("wildcard method", []any{"*"}, true, ""),
			Entry("mixed valid methods", []any{"GET", "*", "POST"}, true, ""),
			Entry("invalid HTTP method", []any{"INVALID"}, false, "invalid HTTP method"),
			Entry("case insensitive methods", []any{"get", "post"}, true, ""), // Should be normalized to uppercase
		)

		It("should inherit base parsing errors", func() {
			permData := map[string]any{
				"reason": "Test reason",
				// Missing allowedUrls
			}

			_, err := ParseHTTPPermissions(permData)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("allowedUrls field is required"))
		})
	})

	Describe("IsRequestAllowed", func() {
		var perms *HTTPPermissions

		Context("HTTP method-specific validation", func() {
			BeforeEach(func() {
				perms = &HTTPPermissions{
					NetworkPermissionsBase: &NetworkPermissionsBase{
						Reason:            "Test permissions",
						AllowLocalNetwork: false,
					},
					AllowedUrls: map[string][]string{
						"https://api.example.com":     {"GET", "POST"},
						"https://upload.example.com":  {"PUT", "PATCH"},
						"https://admin.example.com":   {"DELETE"},
						"https://webhook.example.com": {"*"},
					},
					matcher: NewURLMatcher(),
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
				perms = &HTTPPermissions{
					NetworkPermissionsBase: &NetworkPermissionsBase{
						Reason:            "Test permissions",
						AllowLocalNetwork: false,
					},
					AllowedUrls: map[string][]string{
						"https://api.example.com": {"GET", "POST"}, // Both uppercase for consistency
					},
					matcher: NewURLMatcher(),
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
				perms = &HTTPPermissions{
					NetworkPermissionsBase: &NetworkPermissionsBase{
						Reason:            "Test permissions",
						AllowLocalNetwork: false,
					},
					AllowedUrls: map[string][]string{
						"https://api.example.com/v1/*":     {"GET"},
						"https://api.example.com/v1/users": {"POST", "PUT"},
						"https://*.example.com/public/*":   {"GET", "HEAD"},
						"https://admin.*.example.com":      {"*"},
					},
					matcher: NewURLMatcher(),
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
