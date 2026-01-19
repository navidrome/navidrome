package plugins

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	Describe("UnmarshalJSON", func() {
		It("parses a valid manifest", func() {
			data := []byte(`{
				"name": "Test Plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "A test plugin",
				"website": "https://example.com",
				"permissions": {
					"http": {
						"reason": "Fetch metadata",
						"requiredHosts": ["api.example.com", "*.spotify.com"]
					}
				}
			}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Name).To(Equal("Test Plugin"))
			Expect(m.Author).To(Equal("Test Author"))
			Expect(m.Version).To(Equal("1.0.0"))
			Expect(*m.Description).To(Equal("A test plugin"))
			Expect(*m.Website).To(Equal("https://example.com"))
			Expect(m.Permissions.Http).ToNot(BeNil())
			Expect(*m.Permissions.Http.Reason).To(Equal("Fetch metadata"))
			Expect(m.Permissions.Http.RequiredHosts).To(ContainElements("api.example.com", "*.spotify.com"))
		})

		It("parses a minimal manifest", func() {
			data := []byte(`{
				"name": "Minimal Plugin",
				"author": "Author",
				"version": "1.0.0"
			}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Name).To(Equal("Minimal Plugin"))
			Expect(m.Author).To(Equal("Author"))
			Expect(m.Version).To(Equal("1.0.0"))
			Expect(m.Description).To(BeNil())
			Expect(m.Permissions).To(BeNil())
		})

		It("returns an error for invalid JSON", func() {
			data := []byte(`{invalid json}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when name is missing", func() {
			data := []byte(`{"author": "Test Author", "version": "1.0.0"}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name"))
		})

		It("returns an error when author is missing", func() {
			data := []byte(`{"name": "Test Plugin", "version": "1.0.0"}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("author"))
		})

		It("returns an error when version is missing", func() {
			data := []byte(`{"name": "Test Plugin", "author": "Test Author"}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("version"))
		})

		It("returns an error when name is empty", func() {
			data := []byte(`{"name": "", "author": "Test Author", "version": "1.0.0"}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name"))
		})

		It("returns an error when author is empty", func() {
			data := []byte(`{"name": "Test Plugin", "author": "", "version": "1.0.0"}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("author"))
		})

		It("returns an error when version is empty", func() {
			data := []byte(`{"name": "Test Plugin", "author": "Test Author", "version": ""}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("version"))
		})
	})

	Describe("HasExperimentalThreads", func() {
		It("returns false when no experimental section", func() {
			m := &Manifest{}
			Expect(m.HasExperimentalThreads()).To(BeFalse())
		})

		It("returns false when experimental section has no threads", func() {
			m := &Manifest{
				Experimental: &Experimental{},
			}
			Expect(m.HasExperimentalThreads()).To(BeFalse())
		})

		It("returns true when threads feature is present", func() {
			m := &Manifest{
				Experimental: &Experimental{
					Threads: &ThreadsFeature{},
				},
			}
			Expect(m.HasExperimentalThreads()).To(BeTrue())
		})

		It("returns true when threads feature has a reason", func() {
			reason := "Required for concurrent processing"
			m := &Manifest{
				Experimental: &Experimental{
					Threads: &ThreadsFeature{
						Reason: &reason,
					},
				},
			}
			Expect(m.HasExperimentalThreads()).To(BeTrue())
		})

		It("parses experimental.threads from JSON", func() {
			data := []byte(`{
				"name": "Threaded Plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"experimental": {
					"threads": {
						"reason": "To use multi-threaded WASM module"
					}
				}
			}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.HasExperimentalThreads()).To(BeTrue())
			Expect(m.Experimental.Threads.Reason).ToNot(BeNil())
			Expect(*m.Experimental.Threads.Reason).To(Equal("To use multi-threaded WASM module"))
		})

		It("parses experimental.threads without reason from JSON", func() {
			data := []byte(`{
				"name": "Threaded Plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"experimental": {
					"threads": {}
				}
			}`)

			var m Manifest
			err := json.Unmarshal(data, &m)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.HasExperimentalThreads()).To(BeTrue())
		})
	})

	Describe("ParseManifest", func() {
		It("parses a valid manifest with users permission", func() {
			data := []byte(`{
				"name": "Test Plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"permissions": {
					"subsonicapi": {},
					"users": {}
				}
			}`)

			m, err := ParseManifest(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Name).To(Equal("Test Plugin"))
			Expect(m.Permissions.Subsonicapi).ToNot(BeNil())
			Expect(m.Permissions.Users).ToNot(BeNil())
		})

		It("returns error for invalid JSON", func() {
			data := []byte(`{invalid}`)

			_, err := ParseManifest(data)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when subsonicapi is requested without users permission", func() {
			data := []byte(`{
				"name": "Test Plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"permissions": {
					"subsonicapi": {}
				}
			}`)

			_, err := ParseManifest(data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("subsonicapi"))
			Expect(err.Error()).To(ContainSubstring("users"))
		})
	})

	Describe("Validate", func() {
		It("validates manifest with subsonicapi and users permissions", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
				Permissions: &Permissions{
					Subsonicapi: &SubsonicAPIPermission{},
					Users:       &UsersPermission{},
				},
			}

			err := m.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error when subsonicapi without users permission", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
				Permissions: &Permissions{
					Subsonicapi: &SubsonicAPIPermission{},
				},
			}

			err := m.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("subsonicapi"))
		})

		It("validates manifest without subsonicapi", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
				Permissions: &Permissions{
					Http: &HTTPPermission{},
				},
			}

			err := m.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates manifest without any permissions", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
			}

			err := m.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("ValidateWithCapabilities", func() {
		It("validates scrobbler capability with users permission", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
				Permissions: &Permissions{
					Users: &UsersPermission{},
				},
			}

			err := ValidateWithCapabilities(m, []Capability{CapabilityScrobbler})
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error when scrobbler capability without users permission", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
			}

			err := ValidateWithCapabilities(m, []Capability{CapabilityScrobbler})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scrobbler"))
			Expect(err.Error()).To(ContainSubstring("users"))
		})

		It("validates non-scrobbler capability without users permission", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
			}

			err := ValidateWithCapabilities(m, []Capability{CapabilityMetadataAgent})
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates multiple capabilities including scrobbler", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
				Permissions: &Permissions{
					Users: &UsersPermission{},
				},
			}

			err := ValidateWithCapabilities(m, []Capability{CapabilityMetadataAgent, CapabilityScrobbler})
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates with nil capabilities", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
			}

			err := ValidateWithCapabilities(m, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates with empty capabilities", func() {
			m := &Manifest{
				Name:    "Test",
				Author:  "Author",
				Version: "1.0.0",
			}

			err := ValidateWithCapabilities(m, []Capability{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
