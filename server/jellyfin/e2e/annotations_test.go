package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Annotations", func() {
	BeforeEach(func() { setupTestDB() })

	itemUserData := func(id string) *dto.UserItemDataDto {
		var item dto.BaseItemDto
		parseInto(get("/Items/"+enc(id)), &item)
		return item.UserData
	}

	Describe("favorites", func() {
		It("marks and unmarks an album as favorite", func() {
			id := albumID("Abbey Road")

			var marked dto.UserItemDataDto
			parseInto(post("/Users/admin-1/FavoriteItems/"+enc(id), ""), &marked)
			Expect(marked.IsFavorite).To(BeTrue())
			Expect(itemUserData(id).IsFavorite).To(BeTrue())

			var unmarked dto.UserItemDataDto
			parseInto(del("/Users/admin-1/FavoriteItems/"+enc(id)), &unmarked)
			Expect(unmarked.IsFavorite).To(BeFalse())
			Expect(itemUserData(id).IsFavorite).To(BeFalse())
		})

		It("marks a song as favorite", func() {
			id := songID("So What")
			var data dto.UserItemDataDto
			parseInto(post("/Users/admin-1/FavoriteItems/"+enc(id), ""), &data)
			Expect(itemUserData(id).IsFavorite).To(BeTrue())
		})

		It("marks and unmarks via the current SDK endpoint /UserFavoriteItems/{id} (Jellify)", func() {
			id := songID("Come Together")

			var marked dto.UserItemDataDto
			parseInto(post("/UserFavoriteItems/"+enc(id), ""), &marked)
			Expect(marked.IsFavorite).To(BeTrue())
			Expect(itemUserData(id).IsFavorite).To(BeTrue())

			var unmarked dto.UserItemDataDto
			parseInto(del("/UserFavoriteItems/"+enc(id)), &unmarked)
			Expect(unmarked.IsFavorite).To(BeFalse())
			Expect(itemUserData(id).IsFavorite).To(BeFalse())
		})

		It("filters items to favorites only", func() {
			post("/Users/admin-1/FavoriteItems/"+enc(albumID("Abbey Road")), "")
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&Filters=IsFavorite"))
			Expect(q.TotalRecordCount).To(Equal(1))
			Expect(q.Items[0].Name).To(Equal("Abbey Road"))
		})

		It("marks and lists a playlist as favorite", func() {
			id := createPlaylist("Favorite Mix", nil)
			Expect(post("/Users/admin-1/FavoriteItems/"+enc(id), "").Code).To(Equal(http.StatusOK))
			Expect(itemUserData(id).IsFavorite).To(BeTrue())

			q := queryResult(get("/Items?IncludeItemTypes=Playlist&Recursive=true&Filters=IsFavorite"))
			Expect(q.TotalRecordCount).To(Equal(1))
			Expect(q.Items[0].Name).To(Equal("Favorite Mix"))
		})

		It("filters to favorites via the isFavorite query param (Finamp's artist widget form)", func() {
			// Finamp's "Favourite tracks" widget sends isFavorite=true as a query param (not
			// Filters=IsFavorite), combined with ArtistIds.
			post("/Users/admin-1/FavoriteItems/"+enc(songID("Help!")), "")
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&ArtistIds=" + enc(artistID("The Beatles")) + "&isFavorite=true"))
			Expect(names(q.Items)).To(ConsistOf("Help!"))
		})

		It("returns 404 when favoriting an unknown item", func() {
			Expect(post("/Users/admin-1/FavoriteItems/"+enc("nope"), "").Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("GET /UserItems/{id}/UserData", func() {
		It("returns per-item favorite/played state (Jellify's played/favourite indicators)", func() {
			id := songID("So What")
			post("/Users/admin-1/FavoriteItems/"+enc(id), "")

			var data dto.UserItemDataDto
			parseInto(get("/UserItems/"+enc(id)+"/UserData?userId=admin-1"), &data)
			Expect(data.IsFavorite).To(BeTrue())
			Expect(data.ItemId).To(Equal(enc(id)))
		})

		It("returns a valid (unfavorited) UserData for an item with no annotations", func() {
			var data dto.UserItemDataDto
			parseInto(get("/UserItems/"+enc(albumID("Kind of Blue"))+"/UserData"), &data)
			Expect(data.IsFavorite).To(BeFalse())
			Expect(data.ItemId).To(Equal(enc(albumID("Kind of Blue"))))
		})

		It("returns 404 for an unknown item", func() {
			Expect(get("/UserItems/" + enc("nope") + "/UserData").Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("ratings", func() {
		It("sets and clears an album rating (Jellyfin 0-10 scale)", func() {
			id := albumID("IV")

			var set dto.UserItemDataDto
			parseInto(post("/Users/admin-1/Items/"+enc(id)+"/Rating?Rating=10", ""), &set)
			Expect(set.Rating).ToNot(BeNil())
			Expect(*set.Rating).To(Equal(float64(10)))
			Expect(*itemUserData(id).Rating).To(Equal(float64(10)))

			// Fresh struct: the DELETE response omits the (now-nil) Rating field, so reusing `set`
			// would leave the stale value.
			var cleared dto.UserItemDataDto
			parseInto(del("/Users/admin-1/Items/"+enc(id)+"/Rating"), &cleared)
			Expect(cleared.Rating).To(BeNil())
			Expect(itemUserData(id).Rating).To(BeNil())
		})

		It("sets and reads a playlist rating", func() {
			id := createPlaylist("Rated Mix", nil)
			Expect(post("/Users/admin-1/Items/"+enc(id)+"/Rating?Rating=8", "").Code).To(Equal(http.StatusOK))
			Expect(*itemUserData(id).Rating).To(Equal(float64(8)))
		})

		It("clamps an out-of-range rating to the valid domain", func() {
			id := albumID("Help!")
			var data dto.UserItemDataDto
			parseInto(post("/Users/admin-1/Items/"+enc(id)+"/Rating?Rating=100", ""), &data)
			// 100 clamps to 10 (Jellyfin) -> 5 (Navidrome) -> 10 back out.
			Expect(data.Rating).ToNot(BeNil())
			Expect(*data.Rating).To(Equal(float64(10)))
		})
	})
})
