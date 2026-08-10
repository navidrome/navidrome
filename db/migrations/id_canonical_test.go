package migrations

import (
	"strings"

	"github.com/navidrome/navidrome/model/id"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("canonicalID", func() {
	DescribeTable("transforms each historical id shape",
		func(in, want string) {
			Expect(canonicalID(in)).To(Equal(want))
		},
		Entry("hash-family id (fits 128 bits) is kept", "5cLJPkLA5DK2BADhoeotPk", "5cLJPkLA5DK2BADhoeotPk"),
		Entry("overflowing random id is remapped via md5", "zzzzzzzzzzzzzzzzzzzzzz", "3LyqmwQBm5IRqlVjNYASwb"),
		Entry("legacy 32-hex is re-encoded value-preserving", "e3b7fc2ae9447bbec37a13bf916e3cf6", "6VHl3uR4kss6sUPKA8Cwnk"),
		Entry("playlist uuid is re-encoded value-preserving", "f47ac10b-58cc-4372-a567-0e02b2c3d479", "7rke2SAWaicSeSYzkhww6R"),
		Entry("empty string passes through", "", ""),
		Entry("share id (10 chars) passes through", "aB3xY9kQz1", "aB3xY9kQz1"),
		Entry("truncated Finamp id (16 chars) passes through", "0123456789abcdef", "0123456789abcdef"),
		Entry("22 chars with non-base62 char passes through", "!!!!!!!!!!!!!!!!!!!!!!", "!!!!!!!!!!!!!!!!!!!!!!"),
		Entry("32 chars non-hex passes through", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"),
		Entry("36 chars without uuid dashes passes through", "000000000000000000000000000000000000", "000000000000000000000000000000000000"),
	)

	// The exemptions for participants/tags/folder_ids/similar_artists rest on this invariant.
	It("is the identity on every NewHash id", func() {
		for _, parts := range [][]string{
			{""}, {"a"}, {"The Beatles"}, {"genre", "electronic"},
			{"/music/Artist/Album", "1"}, {strings.Repeat("x", 500)},
		} {
			h := id.NewHash(parts...)
			Expect(h).To(HaveLen(22))
			Expect(canonicalID(h)).To(Equal(h), "NewHash(%v) = %q was rewritten", parts, h)
		}
	})

	It("is idempotent for every shape", func() {
		for _, s := range []string{"5cLJPkLA5DK2BADhoeotPk", "zzzzzzzzzzzzzzzzzzzzzz",
			"e3b7fc2ae9447bbec37a13bf916e3cf6", "f47ac10b-58cc-4372-a567-0e02b2c3d479"} {
			once := canonicalID(s)
			Expect(canonicalID(once)).To(Equal(once))
		}
	})
})
