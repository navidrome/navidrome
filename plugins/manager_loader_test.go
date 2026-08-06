package plugins

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("buildExtismManifest", func() {
	var pkg *ndpPackage

	BeforeEach(func() {
		pkg = &ndpPackage{
			WasmBytes: []byte("wasm"),
			Manifest: &Manifest{Permissions: &Permissions{
				Library: &LibraryPermission{Reason: new("test"), Filesystem: true},
				Http:    &HTTPPermission{Reason: new("test"), RequiredHosts: []string{"example.com"}},
			}},
		}
	})

	It("never sets AllowedPaths, even with filesystem permission", func() {
		Expect(buildExtismManifest(pkg, nil).AllowedPaths).To(BeEmpty())
	})

	It("carries the hosts the plugin is allowed to reach", func() {
		Expect(buildExtismManifest(pkg, nil).AllowedHosts).To(Equal([]string{"example.com"}))
	})
})

var _ = Describe("loadPluginWithConfig", func() {
	// Discovery already rejects these, but a row predating that check, or one
	// left behind by a failed sync, must not reach the mount setup
	It("refuses a plugin whose ID is not usable as a directory name", func() {
		m := &Manager{plugins: make(map[string]*plugin)}

		err := m.loadPluginWithConfig(&model.Plugin{ID: "..", Path: "/does/not/matter.ndp"})

		Expect(err).To(MatchError(ContainSubstring("invalid plugin ID")))
	})
})

var _ = Describe("parsePluginConfig", func() {
	It("returns nil for empty string", func() {
		result, err := parsePluginConfig("")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeNil())
	})

	It("serializes object values as JSON strings", func() {
		result, err := parsePluginConfig(`{"settings": {"enabled": true, "count": 5}}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result["settings"]).To(Equal(`{"count":5,"enabled":true}`))
	})

	It("handles mixed value types", func() {
		result, err := parsePluginConfig(`{"api_key": "secret", "timeout": 30, "rate": 1.5, "enabled": true, "tags": ["a", "b"]}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(5))
		Expect(result["api_key"]).To(Equal("secret"))
		Expect(result["timeout"]).To(Equal("30"))
		Expect(result["rate"]).To(Equal("1.5"))
		Expect(result["enabled"]).To(Equal("true"))
		Expect(result["tags"]).To(Equal(`["a","b"]`))
	})

	It("returns error for invalid JSON", func() {
		_, err := parsePluginConfig(`{invalid json}`)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("parsing plugin config"))
	})

	It("returns error for non-object JSON", func() {
		_, err := parsePluginConfig(`["array", "not", "object"]`)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("parsing plugin config"))
	})

	It("handles null values", func() {
		result, err := parsePluginConfig(`{"key": null}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result["key"]).To(Equal("null"))
	})

	It("handles empty object", func() {
		result, err := parsePluginConfig(`{}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(0))
		Expect(result).ToNot(BeNil())
	})
})
