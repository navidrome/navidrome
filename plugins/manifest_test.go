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
						"allowedHosts": ["api.example.com", "*.spotify.com"]
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
			Expect(m.Permissions.Http.AllowedHosts).To(ContainElements("api.example.com", "*.spotify.com"))
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

	Describe("AllowedHosts", func() {
		It("returns nil when no permissions", func() {
			m := &Manifest{}

			Expect(m.AllowedHosts()).To(BeNil())
		})

		It("returns nil when no HTTP permissions", func() {
			m := &Manifest{
				Permissions: &Permissions{},
			}

			Expect(m.AllowedHosts()).To(BeNil())
		})

		It("returns hosts from permissions", func() {
			m := &Manifest{
				Permissions: &Permissions{
					Http: &HTTPPermission{
						AllowedHosts: []string{"api.example.com", "*.spotify.com"},
					},
				},
			}

			hosts := m.AllowedHosts()
			Expect(hosts).To(Equal([]string{"api.example.com", "*.spotify.com"}))
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
})
