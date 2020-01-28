package persistence

import (
	"context"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlaylistRepository", func() {
	var repo model.PlaylistRepository

	BeforeEach(func() {
		repo = NewPlaylistRepository(context.Background(), orm.NewOrm())
	})

	Describe("Count", func() {
		It("returns the number of playlists in the DB", func() {
			Expect(repo.CountAll()).To(Equal(int64(2)))
		})
	})

	Describe("Exist", func() {
		It("returns true for an existing playlist", func() {
			Expect(repo.Exists("11")).To(BeTrue())
		})
		It("returns false for a non-existing playlist", func() {
			Expect(repo.Exists("666")).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("returns an existing playlist", func() {
			Expect(repo.Get("10")).To(Equal(&plsBest))
		})
		It("returns ErrNotFound for a non-existing playlist", func() {
			_, err := repo.Get("666")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("Put/Get/Delete", func() {
		newPls := model.Playlist{ID: "22", Name: "Great!", Tracks: model.MediaFiles{{ID: "4"}}}
		It("saves the playlist to the DB", func() {
			Expect(repo.Put(&newPls)).To(BeNil())
		})
		It("returns the newly created playlist", func() {
			Expect(repo.Get("22")).To(Equal(&newPls))
		})
		It("returns deletes the playlist", func() {
			Expect(repo.Delete("22")).To(BeNil())
		})
		It("returns error if tries to retrieve the deleted playlist", func() {
			_, err := repo.Get("22")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetWithTracks", func() {
		It("returns an existing playlist", func() {
			pls, err := repo.GetWithTracks("10")
			Expect(err).To(BeNil())
			Expect(pls.Name).To(Equal(plsBest.Name))
			Expect(pls.Tracks).To(Equal(model.MediaFiles{
				songDayInALife,
				songRadioactivity,
			}))
		})
	})

	Describe("GetAll", func() {
		It("returns all playlists from DB", func() {
			Expect(repo.GetAll()).To(Equal(model.Playlists{plsBest, plsCool}))
		})
	})
})
