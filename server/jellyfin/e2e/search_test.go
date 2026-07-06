package e2e

import (
	"fmt"

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

	Describe("pagination totals", func() {
		It("reports the search match count, not the unfiltered library count", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&SearchTerm=Abbey&Limit=50"))
			Expect(q.Items).To(HaveLen(1))
			Expect(q.TotalRecordCount).To(Equal(1)) // not the 5-album library total
		})

		It("reaches the true total when paging song search results", func() {
			// "So" prefix-matches several songs (titles and Solo Artist's tracks); learn the true
			// count from an unpaged query, then walk one-item pages: the reported total must keep
			// the client paging until the last match and stop it exactly there.
			all := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&SearchTerm=So"))
			total := all.TotalRecordCount
			Expect(total).To(Equal(len(all.Items)))
			Expect(total).To(BeNumerically(">=", 2))

			var collected []string
			for start := range total {
				page := queryResult(get(fmt.Sprintf("/Items?IncludeItemTypes=Audio&Recursive=true&SearchTerm=So&Limit=1&StartIndex=%d", start)))
				Expect(page.Items).To(HaveLen(1))
				if start+1 < total {
					Expect(page.TotalRecordCount).To(BeNumerically(">", start+1)) // more remain: keep paging
				} else {
					Expect(page.TotalRecordCount).To(Equal(total)) // last page: exact, so the client stops
				}
				collected = append(collected, page.Items[0].Name)
			}
			Expect(collected).To(ConsistOf(names(all.Items)))
		})
	})
})
