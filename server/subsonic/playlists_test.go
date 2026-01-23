package subsonic

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ core.Playlists = (*fakePlaylists)(nil)

var _ = Describe("buildPlaylist", func() {
	var router *Router
	var ds model.DataStore
	var ctx context.Context
	var playlist model.Playlist

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		ctx = context.Background()

		createdAt := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
		updatedAt := time.Date(2023, 2, 20, 14, 45, 0, 0, time.UTC)

		playlist = model.Playlist{
			ID:        "pls-1",
			Name:      "My Playlist",
			Comment:   "Test comment",
			OwnerName: "admin",
			Public:    true,
			SongCount: 10,
			Duration:  600,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
	})

	Context("with minimal client", func() {
		BeforeEach(func() {
			conf.Server.Subsonic.MinimalClients = "minimal-client"
			player := model.Player{Client: "minimal-client"}
			ctx = request.WithPlayer(ctx, player)
		})

		It("returns only basic fields", func() {
			result := router.buildPlaylist(ctx, playlist)

			Expect(result.Id).To(Equal("pls-1"))
			Expect(result.Name).To(Equal("My Playlist"))
			Expect(result.SongCount).To(Equal(int32(10)))
			Expect(result.Duration).To(Equal(int32(600)))
			Expect(result.Created).To(Equal(playlist.CreatedAt))
			Expect(result.Changed).To(Equal(playlist.UpdatedAt))

			// These should not be set
			Expect(result.Comment).To(BeEmpty())
			Expect(result.Owner).To(BeEmpty())
			Expect(result.Public).To(BeFalse())
			Expect(result.CoverArt).To(BeEmpty())
		})
	})

	Context("with non-minimal client", func() {
		BeforeEach(func() {
			conf.Server.Subsonic.MinimalClients = "minimal-client"
			player := model.Player{Client: "regular-client"}
			ctx = request.WithPlayer(ctx, player)
		})

		It("returns all fields", func() {
			result := router.buildPlaylist(ctx, playlist)

			Expect(result.Id).To(Equal("pls-1"))
			Expect(result.Name).To(Equal("My Playlist"))
			Expect(result.SongCount).To(Equal(int32(10)))
			Expect(result.Duration).To(Equal(int32(600)))
			Expect(result.Created).To(Equal(playlist.CreatedAt))
			Expect(result.Changed).To(Equal(playlist.UpdatedAt))
			Expect(result.Comment).To(Equal("Test comment"))
			Expect(result.Owner).To(Equal("admin"))
			Expect(result.Public).To(BeTrue())
		})
	})

	Context("when minimal clients list is empty", func() {
		BeforeEach(func() {
			conf.Server.Subsonic.MinimalClients = ""
			player := model.Player{Client: "any-client"}
			ctx = request.WithPlayer(ctx, player)
		})

		It("returns all fields", func() {
			result := router.buildPlaylist(ctx, playlist)

			Expect(result.Comment).To(Equal("Test comment"))
			Expect(result.Owner).To(Equal("admin"))
			Expect(result.Public).To(BeTrue())
		})
	})

	Context("when no player in context", func() {
		It("returns all fields", func() {
			result := router.buildPlaylist(ctx, playlist)

			Expect(result.Comment).To(Equal("Test comment"))
			Expect(result.Owner).To(Equal("admin"))
			Expect(result.Public).To(BeTrue())
		})
	})

})

var _ = Describe("UpdatePlaylist", func() {
	var router *Router
	var ds model.DataStore
	var playlists *fakePlaylists

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		playlists = &fakePlaylists{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, playlists, nil, nil, nil, nil)
	})

	It("clears the comment when parameter is empty", func() {
		r := newGetRequest("playlistId=123", "comment=")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastComment).ToNot(BeNil())
		Expect(*playlists.lastComment).To(Equal(""))
	})

	It("leaves comment unchanged when parameter is missing", func() {
		r := newGetRequest("playlistId=123")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastComment).To(BeNil())
	})

	It("sets public to true when parameter is 'true'", func() {
		r := newGetRequest("playlistId=123", "public=true")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastPublic).ToNot(BeNil())
		Expect(*playlists.lastPublic).To(BeTrue())
	})

	It("sets public to false when parameter is 'false'", func() {
		r := newGetRequest("playlistId=123", "public=false")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastPublic).ToNot(BeNil())
		Expect(*playlists.lastPublic).To(BeFalse())
	})

	It("leaves public unchanged when parameter is missing", func() {
		r := newGetRequest("playlistId=123")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastPublic).To(BeNil())
	})
})

type fakePlaylists struct {
	core.Playlists
	lastPlaylistID string
	lastName       *string
	lastComment    *string
	lastPublic     *bool
	lastAdd        []string
	lastRemove     []int
}

func (f *fakePlaylists) Update(ctx context.Context, playlistID string, name *string, comment *string, public *bool, idsToAdd []string, idxToRemove []int) error {
	f.lastPlaylistID = playlistID
	f.lastName = name
	f.lastComment = comment
	f.lastPublic = public
	f.lastAdd = idsToAdd
	f.lastRemove = idxToRemove
	return nil
}
