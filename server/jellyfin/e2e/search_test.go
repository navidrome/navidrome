package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Search with a ParentId library scope is how Finamp drives its search screen. Artists are the
// tricky case: they have no library_id column, so the repo's Search does its own scope handling.
var _ = Describe("Search", func() {
	BeforeEach(func() { setupTestDB() })

	lib1 := func() string { return enc("1") } // Library id 1 encodes to "31"

	Describe("artists", func() {
		It("searches all album artists", func() {
			q := queryResult(get("/Artists/AlbumArtists?SearchTerm=Beatles"))
			Expect(names(q.Items)).To(ConsistOf("The Beatles"))
		})

		It("searches album artists scoped to a library (ParentId)", func() {
			q := queryResult(get("/Artists/AlbumArtists?ParentId=" + lib1() + "&SearchTerm=Beatles&Recursive=true&SortBy=SortName"))
			Expect(names(q.Items)).To(ConsistOf("The Beatles"))
		})

		It("returns an empty result for a non-matching term", func() {
			q := queryResult(get("/Artists?ParentId=" + lib1() + "&SearchTerm=nonexistentxyz"))
			Expect(q.Items).To(BeEmpty())
		})
	})

	Describe("albums and songs", func() {
		It("searches albums scoped to a library", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&ParentId=" + lib1() + "&SearchTerm=Abbey"))
			Expect(names(q.Items)).To(ContainElement("Abbey Road"))
		})

		It("searches songs scoped to a library", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&ParentId=" + lib1() + "&SearchTerm=Stairway"))
			Expect(names(q.Items)).To(ContainElement("Stairway To Heaven"))
		})
	})
})
