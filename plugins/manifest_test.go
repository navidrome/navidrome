package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	Describe("ParseManifest", func() {
		It("parses a valid manifest", func() {
			data := []byte(`{
				"name": "Test Plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "A test plugin",
				"website": "https://example.com",
				"capabilities": ["MetadataAgent"],
				"permissions": {
					"http": {
						"reason": "Fetch metadata",
						"allowedUrls": {
							"https://api.example.com/*": ["GET"]
						}
					}
				}
			}`)

			m, err := ParseManifest(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Name).To(Equal("Test Plugin"))
			Expect(m.Author).To(Equal("Test Author"))
			Expect(m.Version).To(Equal("1.0.0"))
			Expect(m.Description).To(Equal("A test plugin"))
			Expect(m.Website).To(Equal("https://example.com"))
			Expect(m.Capabilities).To(ContainElement(CapabilityMetadataAgent))
			Expect(m.Permissions.HTTP).ToNot(BeNil())
			Expect(m.Permissions.HTTP.Reason).To(Equal("Fetch metadata"))
			Expect(m.Permissions.HTTP.AllowedURLs).To(HaveKey("https://api.example.com/*"))
		})

		It("returns an error for invalid JSON", func() {
			data := []byte(`{invalid json}`)

			_, err := ParseManifest(data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON"))
		})
	})

	Describe("Validate", func() {
		It("returns an error when name is missing", func() {
			m := &Manifest{
				Author:       "Test Author",
				Version:      "1.0.0",
				Capabilities: []Capability{CapabilityMetadataAgent},
			}

			err := m.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name is required"))
		})

		It("returns an error when author is missing", func() {
			m := &Manifest{
				Name:         "Test Plugin",
				Version:      "1.0.0",
				Capabilities: []Capability{CapabilityMetadataAgent},
			}

			err := m.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("author is required"))
		})

		It("returns an error when version is missing", func() {
			m := &Manifest{
				Name:         "Test Plugin",
				Author:       "Test Author",
				Capabilities: []Capability{CapabilityMetadataAgent},
			}

			err := m.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("version is required"))
		})

		It("returns an error when capabilities are missing", func() {
			m := &Manifest{
				Name:    "Test Plugin",
				Author:  "Test Author",
				Version: "1.0.0",
			}

			err := m.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one capability is required"))
		})

		It("returns an error for unknown capability", func() {
			m := &Manifest{
				Name:         "Test Plugin",
				Author:       "Test Author",
				Version:      "1.0.0",
				Capabilities: []Capability{"UnknownCapability"},
			}

			err := m.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown capability"))
		})

		It("returns an error for invalid URL pattern", func() {
			m := &Manifest{
				Name:         "Test Plugin",
				Author:       "Test Author",
				Version:      "1.0.0",
				Capabilities: []Capability{CapabilityMetadataAgent},
				Permissions: Permissions{
					HTTP: &HTTPPermission{
						AllowedURLs: map[string][]string{
							"not-a-valid-url": {"GET"},
						},
					},
				},
			}

			err := m.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid URL pattern"))
		})

		It("validates a valid manifest", func() {
			m := &Manifest{
				Name:         "Test Plugin",
				Author:       "Test Author",
				Version:      "1.0.0",
				Capabilities: []Capability{CapabilityMetadataAgent},
				Permissions: Permissions{
					HTTP: &HTTPPermission{
						AllowedURLs: map[string][]string{
							"https://api.example.com/*": {"GET"},
						},
					},
				},
			}

			err := m.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("HasCapability", func() {
		It("returns true when capability exists", func() {
			m := &Manifest{
				Capabilities: []Capability{CapabilityMetadataAgent},
			}

			Expect(m.HasCapability(CapabilityMetadataAgent)).To(BeTrue())
		})

		It("returns false when capability does not exist", func() {
			m := &Manifest{
				Capabilities: []Capability{},
			}

			Expect(m.HasCapability(CapabilityMetadataAgent)).To(BeFalse())
		})
	})

	Describe("AllowedHosts", func() {
		It("returns nil when no HTTP permissions", func() {
			m := &Manifest{}

			Expect(m.AllowedHosts()).To(BeNil())
		})

		It("returns nil when no allowed URLs", func() {
			m := &Manifest{
				Permissions: Permissions{
					HTTP: &HTTPPermission{},
				},
			}

			Expect(m.AllowedHosts()).To(BeNil())
		})

		It("extracts hosts from URL patterns", func() {
			m := &Manifest{
				Permissions: Permissions{
					HTTP: &HTTPPermission{
						AllowedURLs: map[string][]string{
							"https://api.example.com/*":   {"GET"},
							"https://*.spotify.com/api/*": {"GET"},
						},
					},
				},
			}

			hosts := m.AllowedHosts()
			Expect(hosts).To(ContainElements("api.example.com", "*.spotify.com"))
		})
	})

	Describe("isValidURLPattern", func() {
		DescribeTable("validates URL patterns",
			func(pattern string, expected bool) {
				Expect(isValidURLPattern(pattern)).To(Equal(expected))
			},
			Entry("valid HTTPS URL", "https://api.example.com/path", true),
			Entry("valid HTTP URL", "http://api.example.com/path", true),
			Entry("URL with wildcard in path", "https://api.example.com/*", true),
			Entry("URL with wildcard in host", "https://*.example.com/api/*", true),
			Entry("missing scheme", "api.example.com/path", false),
			Entry("invalid scheme", "ftp://api.example.com/path", false),
			Entry("missing host", "https:///path", false),
		)
	})

	Describe("extractHost", func() {
		DescribeTable("extracts hosts from URL patterns",
			func(pattern string, expected string) {
				Expect(extractHost(pattern)).To(Equal(expected))
			},
			Entry("simple host", "https://api.example.com/path", "api.example.com"),
			Entry("host with wildcard", "https://*.example.com/api/*", "*.example.com"),
			Entry("host with port", "https://api.example.com:8080/path", "api.example.com"),
			Entry("invalid URL", "not-a-url", ""),
		)
	})
})
