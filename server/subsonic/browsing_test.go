package subsonic

import (
	"context"
	"fmt"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func contextWithUser(ctx context.Context, userID string, libraryIDs ...int) context.Context {
	libraries := make([]model.Library, len(libraryIDs))
	for i, id := range libraryIDs {
		libraries[i] = model.Library{ID: id, Name: fmt.Sprintf("Test Library %d", id), Path: fmt.Sprintf("/music/library%d", id)}
	}
	user := model.User{
		ID:        userID,
		Libraries: libraries,
	}
	return request.WithUser(ctx, user)
}

var _ = Describe("Browsing", func() {
	var api *Router
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		api = &Router{ds: ds}
		ctx = context.Background()
	})

	Describe("GetMusicFolders", func() {
		It("should return all libraries the user has access", func() {
			// Create mock user with libraries
			ctx := contextWithUser(ctx, "user-id", 1, 2, 3)

			// Create request
			r := httptest.NewRequest("GET", "/rest/getMusicFolders", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetMusicFolders(r)

			// Verify results
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.MusicFolders).ToNot(BeNil())
			Expect(response.MusicFolders.Folders).To(HaveLen(3))
			Expect(response.MusicFolders.Folders[0].Name).To(Equal("Test Library 1"))
			Expect(response.MusicFolders.Folders[1].Name).To(Equal("Test Library 2"))
			Expect(response.MusicFolders.Folders[2].Name).To(Equal("Test Library 3"))
		})
	})

	Describe("buildFolderDirectory", func() {
		var folder model.Folder

		BeforeEach(func() {
			folder = model.Folder{
				ID:       "folder-1",
				ParentID: "root-1",
				Name:     "Jazz",
			}
		})

		It("returns a directory with correct id, name and parent", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{})

			dir, err := api.buildFolderDirectory(ctx, &folder)
			Expect(err).ToNot(HaveOccurred())
			Expect(dir.Id).To(Equal("folder-1"))
			Expect(dir.Name).To(Equal("Jazz"))
			Expect(dir.Parent).To(Equal("root-1"))
		})

		It("includes child folders as directory entries with IsDir=true", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			childFolder := model.Folder{ID: "folder-2", ParentID: "folder-1", Name: "Blues"}
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{childFolder})

			dir, err := api.buildFolderDirectory(ctx, &folder)
			Expect(err).ToNot(HaveOccurred())
			Expect(dir.Child).To(HaveLen(1))
			Expect(dir.Child[0].Id).To(Equal("folder-2"))
			Expect(dir.Child[0].IsDir).To(BeTrue())
		})

		It("includes media files as directory entries with IsDir=false", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "mf-1", Title: "Track 1", FolderID: "folder-1"},
			})

			dir, err := api.buildFolderDirectory(ctx, &folder)
			Expect(err).ToNot(HaveOccurred())
			Expect(dir.Child).To(HaveLen(1))
			Expect(dir.Child[0].Id).To(Equal("mf-1"))
			Expect(dir.Child[0].IsDir).To(BeFalse())
		})

		It("lists child folders before media files", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			childFolder := model.Folder{ID: "folder-2", ParentID: "folder-1", Name: "Sub"}
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{childFolder})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "mf-1", Title: "Track 1", FolderID: "folder-1"},
			})

			dir, err := api.buildFolderDirectory(ctx, &folder)
			Expect(err).ToNot(HaveOccurred())
			Expect(dir.Child).To(HaveLen(2))
			Expect(dir.Child[0].IsDir).To(BeTrue())
			Expect(dir.Child[1].IsDir).To(BeFalse())
		})

		It("sets cover art when folder has image files", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			folder.ImageFiles = []string{"cover.jpg"}
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{})

			dir, err := api.buildFolderDirectory(ctx, &folder)
			Expect(err).ToNot(HaveOccurred())
			Expect(dir.CoverArt).ToNot(BeEmpty())
		})

		It("returns empty cover art when folder has no image files", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{})

			dir, err := api.buildFolderDirectory(ctx, &folder)
			Expect(err).ToNot(HaveOccurred())
			Expect(dir.CoverArt).To(BeEmpty())
		})
	})

	Describe("GetMusicDirectory with folder ID", func() {
		It("returns directory for a valid folder ID", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			folder := model.Folder{ID: "folder-1", ParentID: "root-1", Name: "Rock"}
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{folder})

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=folder-1", nil)
			r = r.WithContext(ctx)

			response, err := api.GetMusicDirectory(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Directory).ToNot(BeNil())
			Expect(response.Directory.Id).To(Equal("folder-1"))
			Expect(response.Directory.Name).To(Equal("Rock"))
			Expect(response.Directory.Parent).To(Equal("root-1"))
		})

		It("returns error for unknown ID", func() {
			ctx = contextWithUser(ctx, "user-id", 1)

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=nonexistent", nil)
			r = r.WithContext(ctx)

			response, err := api.GetMusicDirectory(r)
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("includes child folders as IsDir=true entries in the directory", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			folder := model.Folder{ID: "folder-1", ParentID: "", Name: "Music"}
			child := model.Folder{ID: "folder-2", ParentID: "folder-1", Name: "Jazz"}
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{folder, child})

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=folder-1", nil)
			r = r.WithContext(ctx)

			response, err := api.GetMusicDirectory(r)
			Expect(err).ToNot(HaveOccurred())
			// The mock returns all folders; verify child-2 is present and IsDir
			Expect(response.Directory.Child).To(ContainElement(
				And(HaveField("Id", "folder-2"), HaveField("IsDir", true)),
			))
		})
	})

	Describe("GetIndexes", func() {
		var lib1 model.Library

		BeforeEach(func() {
			lib1 = model.Library{ID: 1, Name: "Test Library 1", Path: "/music/library1"}
		})

		It("should return error when musicFolderId is not accessible", func() {
			ctx = contextWithUser(ctx, "user-id", 1)

			r := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=2", nil)
			r = r.WithContext(ctx)

			response, err := api.GetIndexes(r)
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("returns an empty index when the library has no top-level folders", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			// Folder repo returns nothing — no top-level folders

			r := httptest.NewRequest("GET", "/rest/getIndexes", nil)
			r = r.WithContext(ctx)

			response, err := api.GetIndexes(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Indexes).ToNot(BeNil())
			Expect(response.Indexes.Index).To(BeEmpty())
		})

		It("returns top-level folders grouped by first letter", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			rootID := model.FolderID(lib1, ".")
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{
				{ID: "f-jazz", ParentID: rootID, Name: "Jazz"},
				{ID: "f-rock", ParentID: rootID, Name: "Rock"},
				{ID: "f-blues", ParentID: rootID, Name: "Blues"},
			})

			r := httptest.NewRequest("GET", "/rest/getIndexes", nil)
			r = r.WithContext(ctx)

			response, err := api.GetIndexes(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Indexes.Index).To(HaveLen(3)) // B, J, R
			// Find the "J" index
			Expect(response.Indexes.Index).To(ContainElement(
				And(
					HaveField("Name", "J"),
					HaveField("Artists", ContainElement(
						And(HaveField("Id", "f-jazz"), HaveField("Name", "Jazz")),
					)),
				),
			))
		})

		It("groups non-alpha folder names under '#'", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			rootID := model.FolderID(lib1, ".")
			ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{
				{ID: "f-90s", ParentID: rootID, Name: "90s Hip-Hop"},
			})

			r := httptest.NewRequest("GET", "/rest/getIndexes", nil)
			r = r.WithContext(ctx)

			response, err := api.GetIndexes(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Indexes.Index).To(HaveLen(1))
			Expect(response.Indexes.Index[0].Name).To(Equal("#"))
			Expect(response.Indexes.Index[0].Artists[0].Id).To(Equal("f-90s"))
		})

		It("falls back to artist-based index when FolderBrowsing is disabled", func() {
			conf.Server.Subsonic.FolderBrowsing = false
			DeferCleanup(func() { conf.Server.Subsonic.FolderBrowsing = true })

			ctx = contextWithUser(ctx, "user-id", 1)
			ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{
				{ID: "a-1", Name: "Kraftwerk"},
			})

			r := httptest.NewRequest("GET", "/rest/getIndexes", nil)
			r = r.WithContext(ctx)

			response, err := api.GetIndexes(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Indexes).ToNot(BeNil())
			// Artist-based index returns artist entries, not folder entries
			Expect(response.Indexes.Index).ToNot(BeEmpty())
			Expect(response.Indexes.Index[0].Artists[0].Id).To(Equal("a-1"))
		})
	})

	Describe("GetArtists", func() {
		It("should validate user access to the specified musicFolderId", func() {
			// Create mock user with access to library 1 only
			ctx = contextWithUser(ctx, "user-id", 1)

			// Create request with musicFolderId=3 (not accessible)
			r := httptest.NewRequest("GET", "/rest/getArtists?musicFolderId=3", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetArtists(r)

			// Should return error due to lack of access
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should default to first accessible library when no musicFolderId specified", func() {
			// Create mock user with access to libraries 1 and 2
			ctx = contextWithUser(ctx, "user-id", 1, 2)

			// Setup minimal mock library data for working tests
			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{
				{ID: 1, Name: "Test Library 1", Path: "/music/library1"},
				{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
			})

			// Setup mock artist data
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{
				{ID: "1", Name: "Test Artist 1"},
				{ID: "2", Name: "Test Artist 2"},
			})

			// Create request without musicFolderId
			r := httptest.NewRequest("GET", "/rest/getArtists", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetArtists(r)

			// Should succeed and use first accessible library (1)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Artist).ToNot(BeNil())
		})
	})
})
