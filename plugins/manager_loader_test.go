//go:build !windows

package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
