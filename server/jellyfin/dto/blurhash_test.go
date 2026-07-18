package dto

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("primaryBlurHash", func() {
	version := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	It("returns the stored hash when it matches the current artwork version", func() {
		Expect(primaryBlurHash("LEHV6nWB2yk8", &version, version)).To(Equal("LEHV6nWB2yk8"))
	})

	It("returns the stored hash when the snapshot is newer than the version (image mtime)", func() {
		newer := version.Add(time.Hour)
		Expect(primaryBlurHash("LEHV6nWB2yk8", &newer, version)).To(Equal("LEHV6nWB2yk8"))
	})

	It("omits when there is no stored hash", func() {
		Expect(primaryBlurHash("", nil, version)).To(BeEmpty())
	})

	It("omits when the stored hash is stale (cover changed, not yet re-served)", func() {
		stale := version.Add(-time.Hour)
		Expect(primaryBlurHash("LEHV6nWB2yk8", &stale, version)).To(BeEmpty())
	})
})

var _ = Describe("primaryBlurHashes", func() {
	It("wraps a hash under the Primary tag", func() {
		Expect(primaryBlurHashes("tag-1", "LEHV6nWB2yk8")).To(
			Equal(map[string]map[string]string{"Primary": {"tag-1": "LEHV6nWB2yk8"}}))
	})

	It("returns nil when there is no hash, so the field is omitted", func() {
		Expect(primaryBlurHashes("tag-1", "")).To(BeNil())
	})
})
