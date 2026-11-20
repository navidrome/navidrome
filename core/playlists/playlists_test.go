package playlists_test

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlists", func() {
	var ds model.DataStore
	var ps playlists.Playlists
	var ctx context.Context
	var songs []string

	smartRules := func() *criteria.Criteria {
		return &criteria.Criteria{Expression: criteria.Contains{"title": "test"}}
	}
	storedTracks := func(id string) []string {
		return trackIDs(storedPlaylist(ctx, ds, id))
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		putUsers(ctx, ds, "123", "user-1", "other-user")
		songs = putTracks(ctx, ds, model.DefaultLibraryID, "a/1.mp3", "a/2.mp3", "a/3.mp3", "a/4.mp3", "a/5.mp3")
		ps = playlists.NewPlaylists(ds, artwork.NewUploader(ds))
		ctx = request.WithUser(ctx, model.User{ID: "123"})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			putPlaylists(ctx, ds, &model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", Public: true})
		})

		isStored := func(id string) bool {
			exists, err := ds.Playlist().Exists(asAdmin(ctx), id)
			Expect(err).ToNot(HaveOccurred())
			return exists
		}

		It("allows owner to delete their playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Delete(ctx, "pls-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(isStored("pls-1")).To(BeFalse())
		})

		It("allows admin to delete any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			err := ps.Delete(ctx, "pls-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(isStored("pls-1")).To(BeFalse())
		})

		It("denies non-owner, non-admin from deleting", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.Delete(ctx, "pls-1")
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(isStored("pls-1")).To(BeTrue())
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Delete(ctx, "nonexistent")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	Describe("Tracks", func() {
		BeforeEach(func() {
			putPlaylists(ctx, ds, &model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"})
			putPlaylists(ctx, ds, &model.Playlist{ID: "pls-123", Name: "Mine", OwnerID: "123"})
			_, err := ds.Playlist().Tracks(ctx, "pls-123", false).Add(ctx, songs[:2])
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the playlist's track repository", func() {
			tracks, err := ps.Tracks(ctx, "pls-123")
			Expect(err).ToNot(HaveOccurred())
			Expect(tracks.GetMediaFileIDs(ctx)).To(Equal(songs[:2]))
		})

		It("returns ErrNotFound for an unknown or invisible playlist", func() {
			_, err := ps.Tracks(ctx, "nonexistent")
			Expect(err).To(MatchError(model.ErrNotFound))
			_, err = ps.Tracks(ctx, "pls-1")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("Create", func() {
		BeforeEach(func() {
			putPlaylists(ctx, ds,
				&model.Playlist{ID: "pls-1", Name: "Existing", OwnerID: "user-1"},
				&model.Playlist{ID: "pls-2", Name: "Other's", OwnerID: "other-user", Public: true},
				&model.Playlist{ID: "pls-smart", Name: "Smart", OwnerID: "user-1", Rules: smartRules()},
				&model.Playlist{ID: "pls-synced", Name: "Synced", OwnerID: "user-1", Sync: true},
				&model.Playlist{ID: "pls-synced-other", Name: "Other's Synced", OwnerID: "other-user", Sync: true, Public: true},
			)
			for _, id := range []string{"pls-1", "pls-2"} {
				_, err := ds.Playlist().Tracks(asAdmin(ctx), id, false).Add(ctx, songs[:2])
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("creates a new playlist with owner set from context", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			id, err := ps.Create(ctx, "", "New Playlist", []string{songs[0], songs[1]})
			Expect(err).ToNot(HaveOccurred())
			Expect(id).ToNot(BeEmpty())
			stored := storedPlaylist(ctx, ds, id)
			Expect(stored.Name).To(Equal("New Playlist"))
			Expect(stored.OwnerID).To(Equal("user-1"))
			Expect(trackIDs(stored)).To(Equal(songs[:2]))
		})

		It("replaces tracks on existing playlist when owner matches", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			id, err := ps.Create(ctx, "pls-1", "", []string{songs[2]})
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal("pls-1"))
			Expect(storedTracks("pls-1")).To(Equal([]string{songs[2]}))
		})

		It("allows admin to replace tracks on any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			id, err := ps.Create(ctx, "pls-2", "", []string{songs[2]})
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal("pls-2"))
			Expect(storedTracks("pls-2")).To(Equal([]string{songs[2]}))
		})

		It("denies non-owner, non-admin from replacing tracks on existing playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.Create(ctx, "pls-2", "", []string{songs[2]})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(storedTracks("pls-2")).To(Equal(songs[:2]))
		})

		It("returns error when existing playlistId not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.Create(ctx, "nonexistent", "", []string{songs[0]})
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("denies replacing tracks on a smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.Create(ctx, "pls-smart", "", []string{songs[0]})
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})

		It("denies replacing tracks on a synced playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.Create(ctx, "pls-synced", "", []string{songs[0]})
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})

		It("denies a non-owner with authorization, not a conflict, on a public synced playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.Create(ctx, "pls-synced-other", "", []string{songs[0]})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			putPlaylists(ctx, ds,
				&model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", Public: true},
				&model.Playlist{ID: "pls-other", Name: "Other's", OwnerID: "other-user"},
				&model.Playlist{ID: "pls-smart", Name: "Smart", OwnerID: "user-1", Rules: smartRules()},
				&model.Playlist{ID: "pls-synced", Name: "Synced", OwnerID: "user-1", Sync: true},
			)
		})

		It("allows owner to update their playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "pls-1", new("Updated Name"), nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist(ctx, ds, "pls-1").Name).To(Equal("Updated Name"))
		})

		It("allows admin to update any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			err := ps.Update(ctx, "pls-other", new("Updated Name"), nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist(ctx, ds, "pls-other").Name).To(Equal("Updated Name"))
		})

		It("denies non-owner, non-admin from updating", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.Update(ctx, "pls-1", new("Updated Name"), nil, nil, nil, nil)
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(storedPlaylist(ctx, ds, "pls-1").Name).To(Equal("My Playlist"))
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "nonexistent", new("Updated Name"), nil, nil, nil, nil)
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("denies adding tracks to a smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "pls-smart", nil, nil, nil, []string{songs[0]}, nil)
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})

		It("denies removing tracks from a smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "pls-smart", nil, nil, nil, nil, []int{0})
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})

		It("allows metadata updates on a smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "pls-smart", new("Updated Smart"), nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist(ctx, ds, "pls-smart").Name).To(Equal("Updated Smart"))
		})

		It("denies adding tracks to a synced playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "pls-synced", nil, nil, nil, []string{songs[0]}, nil)
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})

		It("allows metadata updates on a synced playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.Update(ctx, "pls-synced", new("Renamed Synced"), nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist(ctx, ds, "pls-synced").Name).To(Equal("Renamed Synced"))
		})
	})

	Describe("AddTracks", func() {
		BeforeEach(func() {
			putPlaylists(ctx, ds,
				&model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", Public: true},
				&model.Playlist{ID: "pls-smart", Name: "Smart", OwnerID: "user-1", Rules: smartRules()},
				&model.Playlist{ID: "pls-other", Name: "Other's", OwnerID: "other-user"},
				&model.Playlist{ID: "pls-synced", Name: "Synced", OwnerID: "user-1", Sync: true},
			)
		})

		It("allows owner to add tracks", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			count, err := ps.AddTracks(ctx, "pls-1", []string{songs[0], songs[1]})
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(storedTracks("pls-1")).To(ConsistOf(songs[0], songs[1]))
		})

		It("allows admin to add tracks to any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			count, err := ps.AddTracks(ctx, "pls-other", []string{songs[0]})
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
			Expect(storedTracks("pls-other")).To(Equal([]string{songs[0]}))
		})

		It("denies non-owner, non-admin", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			_, err := ps.AddTracks(ctx, "pls-1", []string{songs[0]})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(storedTracks("pls-1")).To(BeEmpty())
		})

		It("denies editing smart playlists", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.AddTracks(ctx, "pls-smart", []string{songs[0]})
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})

		It("denies editing synced playlists", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.AddTracks(ctx, "pls-synced", []string{songs[0]})
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
			Expect(storedTracks("pls-synced")).To(BeEmpty())
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.AddTracks(ctx, "nonexistent", []string{songs[0]})
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	Describe("RemoveTracks", func() {
		BeforeEach(func() {
			putPlaylists(ctx, ds,
				&model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", Public: true},
				&model.Playlist{ID: "pls-smart", Name: "Smart", OwnerID: "user-1", Rules: smartRules()},
			)
			_, err := ds.Playlist().Tracks(asAdmin(ctx), "pls-1", false).Add(ctx, songs[:3])
			Expect(err).ToNot(HaveOccurred())
		})

		It("allows owner to remove tracks", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveTracks(ctx, "pls-1", []string{"1", "2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(storedTracks("pls-1")).To(Equal([]string{songs[2]}))
		})

		It("denies on smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveTracks(ctx, "pls-smart", []string{"1"})
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})

		It("denies non-owner", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.RemoveTracks(ctx, "pls-1", []string{"1"})
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(storedTracks("pls-1")).To(Equal(songs[:3]))
		})
	})

	Describe("InsertTracks", func() {
		BeforeEach(func() {
			putPlaylists(ctx, ds,
				&model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", Public: true},
				&model.Playlist{ID: "pls-smart", Name: "Smart", OwnerID: "user-1", Rules: smartRules()},
			)
			_, err := ds.Playlist().Tracks(asAdmin(ctx), "pls-1", false).Add(ctx, songs[:3])
			Expect(err).ToNot(HaveOccurred())
		})

		It("inserts the tracks at the given position for the owner", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			count, err := ps.InsertTracks(ctx, "pls-1", []string{songs[3], songs[4]}, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(storedTracks("pls-1")).To(Equal([]string{songs[0], songs[1], songs[3], songs[4], songs[2]}))
		})

		It("denies non-owner, non-admin", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			_, err := ps.InsertTracks(ctx, "pls-1", []string{songs[3]}, 1)
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(storedTracks("pls-1")).To(Equal(songs[:3]))
		})

		It("denies editing smart playlists", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			_, err := ps.InsertTracks(ctx, "pls-smart", []string{songs[0]}, 1)
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})
	})

	Describe("ReorderTrack", func() {
		BeforeEach(func() {
			putPlaylists(ctx, ds,
				&model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
				&model.Playlist{ID: "pls-smart", Name: "Smart", OwnerID: "user-1", Rules: smartRules()},
			)
			_, err := ds.Playlist().Tracks(asAdmin(ctx), "pls-1", false).Add(ctx, songs[:3])
			Expect(err).ToNot(HaveOccurred())
		})

		It("allows owner to reorder", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.ReorderTrack(ctx, "pls-1", 1, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(storedTracks("pls-1")).To(Equal([]string{songs[1], songs[2], songs[0]}))
		})

		It("denies on smart playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.ReorderTrack(ctx, "pls-smart", 1, 3)
			Expect(err).To(MatchError(model.ErrPlaylistNotEditable))
		})
	})

	Describe("SetImage", func() {
		var tmpDir string

		BeforeEach(func() {
			tmpDir = GinkgoT().TempDir()
			conf.Update(func(c *conf.Options) { c.DataFolder = conf.NewDir(tmpDir) })

			putPlaylists(ctx, ds,
				&model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", Public: true},
				&model.Playlist{ID: "pls-other", Name: "Other's", OwnerID: "other-user"},
			)
		})

		It("saves image file and updates UploadedImage", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			reader := strings.NewReader("fake image data")
			err := ps.SetImage(ctx, "pls-1", reader, ".jpg")
			Expect(err).ToNot(HaveOccurred())

			Expect(storedPlaylist(ctx, ds, "pls-1").UploadedImage).To(Equal("pls-1_my_playlist.jpg"))
			absPath := filepath.Join(tmpDir, "artwork", "playlist", "pls-1_my_playlist.jpg")
			data, err := os.ReadFile(absPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("fake image data"))
		})

		It("removes old image when replacing", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})

			// Upload first image
			err := ps.SetImage(ctx, "pls-1", strings.NewReader("first"), ".png")
			Expect(err).ToNot(HaveOccurred())
			oldPath := filepath.Join(tmpDir, "artwork", "playlist", "pls-1_my_playlist.png")
			Expect(oldPath).To(BeAnExistingFile())

			// Upload replacement image
			err = ps.SetImage(ctx, "pls-1", strings.NewReader("second"), ".jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(oldPath).ToNot(BeAnExistingFile())
			newPath := filepath.Join(tmpDir, "artwork", "playlist", "pls-1_my_playlist.jpg")
			Expect(newPath).To(BeAnExistingFile())
		})

		It("allows admin to set image on any playlist", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
			err := ps.SetImage(ctx, "pls-other", strings.NewReader("data"), ".jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist(ctx, ds, "pls-other").UploadedImage).ToNot(BeEmpty())
		})

		It("denies non-owner", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.SetImage(ctx, "pls-1", strings.NewReader("data"), ".jpg")
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(storedPlaylist(ctx, ds, "pls-1").UploadedImage).To(BeEmpty())
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
			tmpDir = GinkgoT().TempDir()
			conf.Update(func(c *conf.Options) { c.DataFolder = conf.NewDir(tmpDir) })

			// Create a real image file on disk
			imgDir := filepath.Join(tmpDir, "artwork", "playlist")
			Expect(os.MkdirAll(imgDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(imgDir, "pls-1.jpg"), []byte("img data"), 0600)).To(Succeed())

			putPlaylists(ctx, ds,
				&model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", Public: true, UploadedImage: "pls-1.jpg"},
				&model.Playlist{ID: "pls-empty", Name: "No Cover", OwnerID: "user-1"},
				&model.Playlist{ID: "pls-other", Name: "Other's", OwnerID: "other-user"},
			)
		})

		It("removes file and clears UploadedImage", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveImage(ctx, "pls-1")
			Expect(err).ToNot(HaveOccurred())

			Expect(storedPlaylist(ctx, ds, "pls-1").UploadedImage).To(BeEmpty())
			absPath := filepath.Join(tmpDir, "artwork", "playlist", "pls-1.jpg")
			Expect(absPath).ToNot(BeAnExistingFile())
		})

		It("succeeds even if playlist has no image", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveImage(ctx, "pls-empty")
			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist(ctx, ds, "pls-empty").UploadedImage).To(BeEmpty())
		})

		It("clears the resolved artwork state and re-queues after removing an upload", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			Expect(ds.Artwork().PutItemArtwork(ctx, &model.ItemArtwork{
				ItemKind: "pl", ItemID: "pls-1", Hash: "oldhash", Source: "upload",
			})).To(Succeed())

			Expect(ps.RemoveImage(ctx, "pls-1")).To(Succeed())

			_, err := ds.Artwork().GetItemArtwork(ctx, model.KindPlaylistArtwork, "pls-1", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))
			queued, _ := ds.ArtworkQueue().DequeueBatch(ctx, 100)
			Expect(queued).To(ContainElement(SatisfyAll(
				HaveField("ItemKind", "pl"),
				HaveField("ItemID", "pls-1"),
				HaveField("Priority", model.ArtworkPriorityBump),
			)))
		})

		It("denies non-owner", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.RemoveImage(ctx, "pls-1")
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(storedPlaylist(ctx, ds, "pls-1").UploadedImage).To(Equal("pls-1.jpg"))
		})

		It("returns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemoveImage(ctx, "nonexistent")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})
})

var _ = Describe("MaxRequestBytes", func() {
	DescribeTable("scales the body limit with DevMaxPlaylistTracksPerRequest",
		func(tracks int, expected int64) {
			conf.Update(func(c *conf.Options) { c.DevMaxPlaylistTracksPerRequest = tracks })
			Expect(playlists.MaxRequestBytes(4<<20, 64)).To(Equal(expected))
		},
		Entry("keeps the floor when the tracks need less", 50_000, int64(4<<20)),
		Entry("grows past the floor when the tracks need more", 100_000, int64(6_400_000)),
		Entry("has no limit when the track limit is 0", 0, int64(math.MaxInt64)),
	)
})
