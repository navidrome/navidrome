package filter_test

import (
	sq "github.com/Masterminds/squirrel"
	. "github.com/navidrome/navidrome/server/subsonic/filter"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SongsByFolder", func() {
	It("sets sort to path", func() {
		opts := SongsByFolder("folder-1")
		Expect(opts.Sort).To(Equal("path"))
	})

	It("filters by the given folder ID", func() {
		opts := SongsByFolder("folder-1")
		Expect(opts.Filters).To(Equal(sq.And{sq.Eq{"missing": false}, sq.Eq{"folder_id": "folder-1"}}))
	})

	It("uses a different folder ID per call", func() {
		opts := SongsByFolder("folder-2")
		Expect(opts.Filters).To(Equal(sq.And{sq.Eq{"missing": false}, sq.Eq{"folder_id": "folder-2"}}))
	})

	It("excludes missing files via default filter", func() {
		opts := SongsByFolder("any-folder")
		filters, ok := opts.Filters.(sq.And)
		Expect(ok).To(BeTrue())
		Expect(filters).To(ContainElement(sq.Eq{"missing": false}))
	})
})
