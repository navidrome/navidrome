package dto

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("blurHash", func() {
	It("returns a 6-char valid blurhash starting with the 1x1 component prefix", func() {
		h := blurHash("x")
		Expect(h).To(HaveLen(6))
		Expect(h).To(HavePrefix("00"))
		for _, c := range h {
			Expect(strings.ContainsRune(base83Alphabet, c)).To(BeTrue(), "unexpected char %q", c)
		}
	})

	It("is deterministic for the same seed", func() {
		Expect(blurHash("cover-tag-1")).To(Equal(blurHash("cover-tag-1")))
	})

	It("differs for different seeds", func() {
		Expect(blurHash("cover-tag-1")).ToNot(Equal(blurHash("cover-tag-2")))
	})
})

var _ = Describe("primaryBlurHash", func() {
	version := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	It("returns the stored hash when it matches the current artwork version", func() {
		Expect(primaryBlurHash("LEHV6nWB2yk8", &version, "id-1", version)).To(Equal("LEHV6nWB2yk8"))
	})

	It("returns the stored hash when the snapshot is newer than the version (image mtime)", func() {
		newer := version.Add(time.Hour)
		Expect(primaryBlurHash("LEHV6nWB2yk8", &newer, "id-1", version)).To(Equal("LEHV6nWB2yk8"))
	})

	It("falls back to a fake when there is no stored hash", func() {
		h := primaryBlurHash("", nil, "id-1", version)
		Expect(h).To(HaveLen(6))
	})

	It("falls back to a fake when the stored hash is stale", func() {
		stale := version.Add(-time.Hour)
		h := primaryBlurHash("LEHV6nWB2yk8", &stale, "id-1", version)
		Expect(h).To(HaveLen(6))
		Expect(h).ToNot(Equal("LEHV6nWB2yk8"))
	})

	It("rotates the fake when the artwork version moves", func() {
		h1 := primaryBlurHash("", nil, "id-1", version)
		h2 := primaryBlurHash("", nil, "id-1", version.Add(time.Hour))
		Expect(h1).ToNot(Equal(h2))
	})
})
