package persistence

import (
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlaylistTrackRepository", func() {
	var repo model.PlaylistTrackRepository

	BeforeEach(func() {
		ctx := log.NewContext(GinkgoT().Context())
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
		repo = NewPlaylistRepository(ctx, GetDBXBuilder()).Tracks(plsBest.ID, true)
	})

	Describe("GetCursor", func() {
		It("yields the same tracks as GetAll", func() {
			opts := model.QueryOptions{Sort: "id"}
			want, err := repo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(want).To(HaveLen(2))

			Expect(collectCursor(repo.GetCursor(opts))).To(Equal([]model.PlaylistTrack(want)))
		})

		It("honors Max and Offset", func() {
			opts := model.QueryOptions{Sort: "id", Max: 1, Offset: 1}
			want, err := repo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(want).To(HaveLen(1))

			Expect(collectCursor(repo.GetCursor(opts))).To(Equal([]model.PlaylistTrack(want)))
		})
	})

	Describe("CountAll", func() {
		It("returns the number of tracks in the playlist", func() {
			Expect(repo.CountAll()).To(Equal(int64(2)))
		})

		It("ignores Max and Offset", func() {
			Expect(repo.CountAll(model.QueryOptions{Max: 1, Offset: 1})).To(Equal(int64(2)))
		})
	})

	Describe("GetMediaFileIDs", func() {
		It("returns the song ids in playlist order", func() {
			Expect(repo.GetMediaFileIDs(model.QueryOptions{Sort: "id"})).
				To(Equal([]string{songDayInALife.ID, songRadioactivity.ID}))
		})

		It("honors Max and Offset", func() {
			Expect(repo.GetMediaFileIDs(model.QueryOptions{Sort: "id", Max: 1, Offset: 1})).
				To(Equal([]string{songRadioactivity.ID}))
		})
	})
})
