package playlists_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlists", func() {
	var ds *tests.MockDataStore
	var ps playlists.Playlists
	var mockPlsRepo *tests.MockPlaylistRepo
	ctx := context.Background()

	BeforeEach(func() {
		mockPlsRepo = tests.CreateMockPlaylistRepo()
		ds = &tests.MockDataStore{
			MockedPlaylist: mockPlsRepo,
			MockedLibrary:  &tests.MockLibraryRepo{},
		}
		ctx = request.WithUser(ctx, model.User{ID: "123"})
	})

	Describe("Delete", func() {
		var mockTracks *tests.MockPlaylistTrackRepo

		BeforeEach(func() {
			mockTracks = &tests.MockPlaylistTrackRepo{AddCount: 3}
			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1": {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
			}
			mockPlsRepo.TracksRepo = mockTracks
			ps = playlists.NewPlaylists(ds)
		})

		It("allows owner to delete their playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Delete(ctx, "pls-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(mockPlsRepo.Deleted).To(ContainElement("pls-1"))
		})

		It("allows admin to delete any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			err := ps.Delete(ctx, "pls-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(mockPlsRepo.Deleted).To(ContainElement("pls-1"))
		})

		It("denies non-owner, non-admin from deleting", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.Delete(ctx, "pls-1")
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(mockPlsRepo.Deleted).To(BeEmpty())
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Delete(ctx, "nonexistent")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	Describe("Create", func() {
		BeforeEach(func() {
			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1": {ID: "pls-1", Name: "Existing", OwnerID: "user-1"},
				"pls-2": {ID: "pls-2", Name: "Other's", OwnerID: "other-user"},
				"pls-smart": {ID: "pls-smart", Name: "Smart", OwnerID: "user-1",
					Rules: &criteria.Criteria{Expression: criteria.Contains{"title": "test"}}},
			}
			ps = playlists.NewPlaylists(ds)
		})

		It("creates a new playlist with owner set from context", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			id, err := ps.Create(ctx, "", "New Playlist", []string{"song-1", "song-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(id).ToNot(BeEmpty())
			Expect(mockPlsRepo.Last.Name).To(Equal("New Playlist"))
			Expect(mockPlsRepo.Last.OwnerID).To(Equal("user-1"))
		})

		It("replaces tracks on existing playlist when owner matches", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			id, err := ps.Create(ctx, "pls-1", "", []string{"song-3"})
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal("pls-1"))
			Expect(mockPlsRepo.Last.Tracks).To(HaveLen(1))
		})

		It("allows admin to replace tracks on any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			id, err := ps.Create(ctx, "pls-2", "", []string{"song-3"})
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal("pls-2"))
		})

		It("denies non-owner, non-admin from replacing tracks on existing playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.Create(ctx, "pls-2", "", []string{"song-3"})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("returns error when existing playlistId not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.Create(ctx, "nonexistent", "", []string{"song-1"})
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("denies replacing tracks on a smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.Create(ctx, "pls-smart", "", []string{"song-1"})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})
	})

	Describe("Update", func() {
		var mockTracks *tests.MockPlaylistTrackRepo

		BeforeEach(func() {
			mockTracks = &tests.MockPlaylistTrackRepo{AddCount: 2}
			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1":     {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
				"pls-other": {ID: "pls-other", Name: "Other's", OwnerID: "other-user"},
				"pls-smart": {ID: "pls-smart", Name: "Smart", OwnerID: "user-1",
					Rules: &criteria.Criteria{Expression: criteria.Contains{"title": "test"}}},
			}
			mockPlsRepo.TracksRepo = mockTracks
			ps = playlists.NewPlaylists(ds)
		})

		It("allows owner to update their playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			newName := "Updated Name"
			err := ps.Update(ctx, "pls-1", &newName, nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("allows admin to update any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			newName := "Updated Name"
			err := ps.Update(ctx, "pls-other", &newName, nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("denies non-owner, non-admin from updating", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			newName := "Updated Name"
			err := ps.Update(ctx, "pls-1", &newName, nil, nil, nil, nil)
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			newName := "Updated Name"
			err := ps.Update(ctx, "nonexistent", &newName, nil, nil, nil, nil)
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("denies adding tracks to a smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "pls-smart", nil, nil, nil, []string{"song-1"}, nil)
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("denies removing tracks from a smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "pls-smart", nil, nil, nil, nil, []int{0})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("allows metadata updates on a smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			newName := "Updated Smart"
			err := ps.Update(ctx, "pls-smart", &newName, nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("AddTracks", func() {
		var mockTracks *tests.MockPlaylistTrackRepo

		BeforeEach(func() {
			mockTracks = &tests.MockPlaylistTrackRepo{AddCount: 2}
			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1": {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
				"pls-smart": {ID: "pls-smart", Name: "Smart", OwnerID: "user-1",
					Rules: &criteria.Criteria{Expression: criteria.Contains{"title": "test"}}},
				"pls-other": {ID: "pls-other", Name: "Other's", OwnerID: "other-user"},
			}
			mockPlsRepo.TracksRepo = mockTracks
			ps = playlists.NewPlaylists(ds)
		})

		It("allows owner to add tracks", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			count, err := ps.AddTracks(ctx, "pls-1", []string{"song-1", "song-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(mockTracks.AddedIds).To(ConsistOf("song-1", "song-2"))
		})

		It("allows admin to add tracks to any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			count, err := ps.AddTracks(ctx, "pls-other", []string{"song-1"})
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
		})

		It("denies non-owner, non-admin", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			_, err := ps.AddTracks(ctx, "pls-1", []string{"song-1"})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("denies editing smart playlists", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.AddTracks(ctx, "pls-smart", []string{"song-1"})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.AddTracks(ctx, "nonexistent", []string{"song-1"})
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	Describe("RemoveTracks", func() {
		var mockTracks *tests.MockPlaylistTrackRepo

		BeforeEach(func() {
			mockTracks = &tests.MockPlaylistTrackRepo{}
			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1": {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
				"pls-smart": {ID: "pls-smart", Name: "Smart", OwnerID: "user-1",
					Rules: &criteria.Criteria{Expression: criteria.Contains{"title": "test"}}},
			}
			mockPlsRepo.TracksRepo = mockTracks
			ps = playlists.NewPlaylists(ds)
		})

		It("allows owner to remove tracks", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveTracks(ctx, "pls-1", []string{"track-1", "track-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(mockTracks.DeletedIds).To(ConsistOf("track-1", "track-2"))
		})

		It("denies on smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveTracks(ctx, "pls-smart", []string{"track-1"})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("denies non-owner", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.RemoveTracks(ctx, "pls-1", []string{"track-1"})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})
	})

	Describe("ReorderTrack", func() {
		var mockTracks *tests.MockPlaylistTrackRepo

		BeforeEach(func() {
			mockTracks = &tests.MockPlaylistTrackRepo{}
			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1": {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
				"pls-smart": {ID: "pls-smart", Name: "Smart", OwnerID: "user-1",
					Rules: &criteria.Criteria{Expression: criteria.Contains{"title": "test"}}},
			}
			mockPlsRepo.TracksRepo = mockTracks
			ps = playlists.NewPlaylists(ds)
		})

		It("allows owner to reorder", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.ReorderTrack(ctx, "pls-1", 1, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(mockTracks.Reordered).To(BeTrue())
		})

		It("denies on smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.ReorderTrack(ctx, "pls-smart", 1, 3)
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})
	})

	Describe("SetImage", func() {
		var tmpDir string

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			tmpDir = GinkgoT().TempDir()
			conf.Server.DataFolder = tmpDir

			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1":     {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
				"pls-other": {ID: "pls-other", Name: "Other's", OwnerID: "other-user"},
			}
			ps = playlists.NewPlaylists(ds)
		})

		It("saves image file and updates ImageFile", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			reader := strings.NewReader("fake image data")
			err := ps.SetImage(ctx, "pls-1", reader, ".jpg")
			Expect(err).ToNot(HaveOccurred())

			Expect(mockPlsRepo.Last.ImageFile).To(Equal("pls-1.jpg"))
			absPath := filepath.Join(tmpDir, "artwork", "playlist", "pls-1.jpg")
			data, err := os.ReadFile(absPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("fake image data"))
		})

		It("removes old image when replacing", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})

			// Upload first image
			err := ps.SetImage(ctx, "pls-1", strings.NewReader("first"), ".png")
			Expect(err).ToNot(HaveOccurred())
			oldPath := filepath.Join(tmpDir, "artwork", "playlist", "pls-1.png")
			Expect(oldPath).To(BeAnExistingFile())

			// Upload replacement image
			err = ps.SetImage(ctx, "pls-1", strings.NewReader("second"), ".jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(oldPath).ToNot(BeAnExistingFile())
			newPath := filepath.Join(tmpDir, "artwork", "playlist", "pls-1.jpg")
			Expect(newPath).To(BeAnExistingFile())
		})

		It("allows admin to set image on any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			err := ps.SetImage(ctx, "pls-other", strings.NewReader("data"), ".jpg")
			Expect(err).ToNot(HaveOccurred())
		})

		It("denies non-owner", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.SetImage(ctx, "pls-1", strings.NewReader("data"), ".jpg")
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.SetImage(ctx, "nonexistent", strings.NewReader("data"), ".jpg")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	Describe("RemoveImage", func() {
		var tmpDir string

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			tmpDir = GinkgoT().TempDir()
			conf.Server.DataFolder = tmpDir

			// Create a real image file on disk
			imgDir := filepath.Join(tmpDir, "artwork", "playlist")
			Expect(os.MkdirAll(imgDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(imgDir, "pls-1.jpg"), []byte("img data"), 0600)).To(Succeed())

			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1":     {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", ImageFile: "pls-1.jpg"},
				"pls-empty": {ID: "pls-empty", Name: "No Cover", OwnerID: "user-1"},
				"pls-other": {ID: "pls-other", Name: "Other's", OwnerID: "other-user"},
			}
			ps = playlists.NewPlaylists(ds)
		})

		It("removes file and clears ImageFile", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveImage(ctx, "pls-1")
			Expect(err).ToNot(HaveOccurred())

			Expect(mockPlsRepo.Last.ImageFile).To(BeEmpty())
			absPath := filepath.Join(tmpDir, "artwork", "playlist", "pls-1.jpg")
			Expect(absPath).ToNot(BeAnExistingFile())
		})

		It("succeeds even if playlist has no image", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveImage(ctx, "pls-empty")
			Expect(err).ToNot(HaveOccurred())
			Expect(mockPlsRepo.Last.ImageFile).To(BeEmpty())
		})

		It("denies non-owner", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.RemoveImage(ctx, "pls-1")
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveImage(ctx, "nonexistent")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})
})
