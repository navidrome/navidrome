package subsonic

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockFolderRepo struct {
	model.FolderRepository
	folders               map[string]*model.Folder
	byPath                map[string]*model.Folder
	getRootSubfoldersFunc func(libraryIDs ...int) ([]model.Folder, error)
	getSubfoldersFunc     func(parentID string, libraryIDs ...int) ([]model.Folder, error)
}

func newMockFolderRepo() *mockFolderRepo {
	return &mockFolderRepo{
		folders: make(map[string]*model.Folder),
		byPath:  make(map[string]*model.Folder),
	}
}

func (m *mockFolderRepo) SetFolders(folders ...model.Folder) {
	for i, f := range folders {
		m.folders[f.ID] = &folders[i]
	}
}

func (m *mockFolderRepo) SetByPath(lib model.Library, p string, folder *model.Folder) {
	key := fmt.Sprintf("%d:%s", lib.ID, p)
	m.byPath[key] = folder
}

func (m *mockFolderRepo) Get(id string) (*model.Folder, error) {
	if f, ok := m.folders[id]; ok {
		return f, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockFolderRepo) GetByPath(lib model.Library, p string) (*model.Folder, error) {
	key := fmt.Sprintf("%d:%s", lib.ID, p)
	if f, ok := m.byPath[key]; ok {
		return f, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockFolderRepo) GetRootSubfoldersWithAudio(libraryIDs ...int) ([]model.Folder, error) {
	if m.getRootSubfoldersFunc != nil {
		return m.getRootSubfoldersFunc(libraryIDs...)
	}
	var res []model.Folder
	for _, f := range m.folders {
		if (f.ParentID == "" || f.ParentID == "root-1" || f.ParentID == "root-2" || f.ParentID == "root-3") && f.Name != "." && !f.Missing {
			if len(libraryIDs) == 0 || slices.Contains(libraryIDs, f.LibraryID) {
				res = append(res, *f)
			}
		}
	}
	slices.SortFunc(res, func(a, b model.Folder) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return res, nil
}

func (m *mockFolderRepo) GetSubfoldersWithAudio(parentID string, libraryIDs ...int) ([]model.Folder, error) {
	if m.getSubfoldersFunc != nil {
		return m.getSubfoldersFunc(parentID, libraryIDs...)
	}
	var res []model.Folder
	for _, f := range m.folders {
		if f.ParentID == parentID && !f.Missing {
			if len(libraryIDs) == 0 || slices.Contains(libraryIDs, f.LibraryID) {
				res = append(res, *f)
			}
		}
	}
	slices.SortFunc(res, func(a, b model.Folder) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return res, nil
}

func (m *mockFolderRepo) GetCoverArtForFolders(folderIDs ...string) (map[string]string, error) {
	res := make(map[string]string)
	for _, id := range folderIDs {
		if f, ok := m.folders[id]; ok && f.NumAudioFiles > 0 {
			res[id] = "al-" + id
		}
	}
	return res, nil
}

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
	var folderRepo *mockFolderRepo

	BeforeEach(func() {
		mockDS := &tests.MockDataStore{}
		folderRepo = newMockFolderRepo()
		mockDS.MockedFolder = folderRepo
		ds = mockDS
		auth.Init(ds)
		api = &Router{ds: ds}
		ctx = context.Background()

		mockMF := ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
		mockMF.GetAllFn = func(qo ...model.QueryOptions) (model.MediaFiles, error) {
			if len(qo) > 0 && qo[0].Filters != nil {
				if and, ok := qo[0].Filters.(squirrel.And); ok {
					for _, cond := range and {
						if eq, ok := cond.(squirrel.Eq); ok {
							if folderID, ok := eq["media_file.folder_id"].(string); ok {
								var res model.MediaFiles
								for _, mf := range mockMF.Data {
									if mf.FolderID == folderID && !mf.Missing {
										res = append(res, *mf)
									}
								}
								slices.SortFunc(res, func(a, b model.MediaFile) int {
									return cmp.Or(
										cmp.Compare(a.DiscNumber, b.DiscNumber),
										cmp.Compare(a.TrackNumber, b.TrackNumber),
										cmp.Compare(a.Title, b.Title),
									)
								})
								return res, nil
							}
						}
					}
				}
				if eq, ok := qo[0].Filters.(squirrel.Eq); ok {
					if albumID, ok := eq["album_id"].(string); ok {
						var res model.MediaFiles
						for _, mf := range mockMF.Data {
							if mf.AlbumID == albumID && !mf.Missing {
								res = append(res, *mf)
							}
						}
						return res, nil
					}
				}
				if and, ok := qo[0].Filters.(squirrel.And); ok {
					for _, cond := range and {
						if eq, ok := cond.(squirrel.Eq); ok {
							if albumID, ok := eq["album_id"].(string); ok {
								var res model.MediaFiles
								for _, mf := range mockMF.Data {
									if mf.AlbumID == albumID && !mf.Missing {
										res = append(res, *mf)
									}
								}
								return res, nil
							}
						}
					}
				}
			}
			var res model.MediaFiles
			for _, mf := range mockMF.Data {
				res = append(res, *mf)
			}
			slices.SortFunc(res, func(a, b model.MediaFile) int {
				return cmp.Compare(a.ID, b.ID)
			})
			return res, nil
		}
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

	Describe("GetIndexes", func() {
		It("should validate user access to the specified musicFolderId", func() {
			// Create mock user with access to library 1 only
			ctx = contextWithUser(ctx, "user-id", 1)

			// Create request with musicFolderId=2 (not accessible)
			r := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=2", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetIndexes(r)

			// Should return error due to lack of access
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should default to first accessible library when no musicFolderId specified", func() {
			// Create mock user with access to libraries 2 and 3
			ctx = contextWithUser(ctx, "user-id", 2, 3)

			// Setup minimal mock library data for working tests
			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{
				{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
				{ID: 3, Name: "Test Library 3", Path: "/music/library3"},
			})

			// Setup mock artist data
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{
				{ID: "1", Name: "Test Artist 1"},
				{ID: "2", Name: "Test Artist 2"},
			})

			// Create request without musicFolderId
			r := httptest.NewRequest("GET", "/rest/getIndexes", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetIndexes(r)

			// Should succeed and use first accessible library (2)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Indexes).ToNot(BeNil())
		})

		It("returns top-level folders grouped alphabetically into res.Indexes.Index with artist elements", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			lib1 := model.Library{ID: 1, Name: "Test Library 1", Path: "/music/library1"}

			rootFolder := &model.Folder{ID: "root-1", LibraryID: 1, Path: "", Name: "."}
			folderRepo.SetFolders(
				*rootFolder,
				model.Folder{ID: "f-beatles", LibraryID: 1, Path: "", Name: "The Beatles", ParentID: "root-1"},
				model.Folder{ID: "f-pop", LibraryID: 1, Path: "", Name: "Pop", ParentID: "root-1"},
				model.Folder{ID: "f-1984", LibraryID: 1, Path: "", Name: "1984", ParentID: "root-1"},
			)
			folderRepo.SetByPath(lib1, ".", rootFolder)

			r := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=1", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetIndexes(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Indexes).ToNot(BeNil())

			indexNames := slice.Map(resp.Indexes.Index, func(idx responses.Index) string { return idx.Name })
			Expect(indexNames).To(Equal([]string{"#", "B", "P"}))

			Expect(resp.Indexes.Index[0].Artists).To(ConsistOf(responses.Artist{Id: "f-1984", Name: "1984"}))
			Expect(resp.Indexes.Index[1].Artists).To(ConsistOf(responses.Artist{Id: "f-beatles", Name: "The Beatles"}))
			Expect(resp.Indexes.Index[2].Artists).To(ConsistOf(responses.Artist{Id: "f-pop", Name: "Pop"}))
		})

		It("returns loose root tracks in res.Indexes.Child", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			lib1 := model.Library{ID: 1, Name: "Test Library 1", Path: "/music/library1"}

			rootFolder := &model.Folder{ID: "root-1", LibraryID: 1, Path: "", Name: "."}
			folderRepo.SetFolders(*rootFolder)
			folderRepo.SetByPath(lib1, ".", rootFolder)

			mockMF := ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
			mockMF.SetData(model.MediaFiles{
				{ID: "track-root-1", Title: "Root Song 1", FolderID: "root-1", TrackNumber: 1},
				{ID: "track-root-2", Title: "Root Song 2", FolderID: "root-1", TrackNumber: 2},
				{ID: "track-sub", Title: "Sub Song", FolderID: "sub-folder", TrackNumber: 1},
			})

			r := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=1", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetIndexes(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Indexes).ToNot(BeNil())
			Expect(resp.Indexes.Child).To(HaveLen(2))
			Expect(resp.Indexes.Child[0].Id).To(Equal("track-root-1"))
			Expect(resp.Indexes.Child[0].Title).To(Equal("Root Song 1"))
			Expect(resp.Indexes.Child[0].IsDir).To(BeFalse())
			Expect(resp.Indexes.Child[1].Id).To(Equal("track-root-2"))
			Expect(resp.Indexes.Child[1].Title).To(Equal("Root Song 2"))
		})

		It("honors musicFolderId query parameter", func() {
			ctx = contextWithUser(ctx, "user-id", 1, 2)
			lib1 := model.Library{ID: 1, Name: "Test Library 1", Path: "/music/library1"}
			lib2 := model.Library{ID: 2, Name: "Test Library 2", Path: "/music/library2"}

			rootFolder1 := &model.Folder{ID: "root-1", LibraryID: 1, Path: "", Name: "."}
			rootFolder2 := &model.Folder{ID: "root-2", LibraryID: 2, Path: "", Name: "."}
			folderRepo.SetFolders(
				*rootFolder1,
				*rootFolder2,
				model.Folder{ID: "f-lib1", LibraryID: 1, Path: "", Name: "Alpha", ParentID: "root-1"},
				model.Folder{ID: "f-lib2", LibraryID: 2, Path: "", Name: "Beta", ParentID: "root-2"},
			)
			folderRepo.SetByPath(lib1, ".", rootFolder1)
			folderRepo.SetByPath(lib2, ".", rootFolder2)

			// Query musicFolderId=1 only
			r1 := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=1", nil)
			r1 = r1.WithContext(ctx)
			resp1, err := api.GetIndexes(r1)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp1.Indexes.Index).To(HaveLen(1))
			Expect(resp1.Indexes.Index[0].Name).To(Equal("A"))
			Expect(resp1.Indexes.Index[0].Artists[0].Name).To(Equal("Alpha"))

			// Query musicFolderId=2 only
			r2 := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=2", nil)
			r2 = r2.WithContext(ctx)
			resp2, err := api.GetIndexes(r2)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp2.Indexes.Index).To(HaveLen(1))
			Expect(resp2.Indexes.Index[0].Name).To(Equal("B"))
			Expect(resp2.Indexes.Index[0].Artists[0].Name).To(Equal("Beta"))
		})

		It("returns empty response without indexes when ifModifiedSince is after last scan timestamp", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			lib1 := model.Library{ID: 1, Name: "Test Library 1", Path: "/music/library1"}
			rootFolder := &model.Folder{ID: "root-1", LibraryID: 1, Path: "", Name: "."}
			folderRepo.SetFolders(
				*rootFolder,
				model.Folder{ID: "f-beatles", LibraryID: 1, Path: "", Name: "The Beatles", ParentID: "root-1"},
			)
			folderRepo.SetByPath(lib1, ".", rootFolder)

			scanTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
			_ = ds.Property(ctx).Put(consts.LastScanStartTimeKey, scanTime.Format(time.RFC3339))

			ifModifiedSince := scanTime.Add(time.Hour).UnixMilli()
			r := httptest.NewRequest("GET", fmt.Sprintf("/rest/getIndexes?musicFolderId=1&ifModifiedSince=%d", ifModifiedSince), nil)
			r = r.WithContext(ctx)

			resp, err := api.GetIndexes(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Indexes).To(BeNil())
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

		It("still returns tag-based artist index unchanged", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{
				{ID: "artist-1", Name: "Pink Floyd"},
			})

			r := httptest.NewRequest("GET", "/rest/getArtists", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetArtists(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Artist).ToNot(BeNil())
			Expect(resp.Artist.Index).ToNot(BeEmpty())
			Expect(resp.Artist.Index[0].Artists[0].Name).To(Equal("Pink Floyd"))
		})
	})

	Describe("GetMusicDirectory", func() {
		It("returns directory with subfolders and tracks when called with folder ID", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			parentFolder := &model.Folder{ID: "f-rock", LibraryID: 1, Name: "Rock", ParentID: "root-1"}
			subFolder := model.Folder{ID: "f-beatles", LibraryID: 1, Name: "The Beatles", ParentID: "f-rock"}
			folderRepo.SetFolders(*parentFolder, subFolder)

			mockMF := ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
			mockMF.SetData(model.MediaFiles{
				{ID: "song-rock-1", Title: "Rock Song 1", FolderID: "f-rock", TrackNumber: 1},
			})

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=f-rock", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Directory).ToNot(BeNil())
			Expect(resp.Directory.Id).To(Equal("f-rock"))
			Expect(resp.Directory.Name).To(Equal("Rock"))
			Expect(resp.Directory.Parent).To(Equal("root-1"))
			Expect(resp.Directory.Child).To(HaveLen(2))
			Expect(resp.Directory.Child[0].Id).To(Equal("f-beatles"))
			Expect(resp.Directory.Child[0].IsDir).To(BeTrue())
			Expect(resp.Directory.Child[1].Id).To(Equal("song-rock-1"))
			Expect(resp.Directory.Child[1].IsDir).To(BeFalse())
		})

		It("populates coverArt for directory and child subfolders", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
			mockAlbumRepo.SetData(model.Albums{
				{ID: "alb-rock", Name: "Rock Album", ItemImage: model.ItemImage{ImageHash: "rockhash"}},
			})

			parentFolder := &model.Folder{ID: "f-artist", LibraryID: 1, Name: "10,000 Maniacs", ParentID: "root-1"}
			subFolder := model.Folder{ID: "f-album", LibraryID: 1, Name: "MTV Unplugged", ParentID: "f-artist", NumAudioFiles: 10}
			folderRepo.SetFolders(*parentFolder, subFolder)

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=f-artist", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Directory).ToNot(BeNil())
			Expect(resp.Directory.Child).To(HaveLen(1))
			Expect(resp.Directory.Child[0].Id).To(Equal("f-album"))
			Expect(resp.Directory.Child[0].IsDir).To(BeTrue())
			Expect(resp.Directory.Child[0].CoverArt).To(Equal("al-f-album"))

			// Now test requesting the album folder directly
			mockMF := ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
			mockMF.SetData(model.MediaFiles{
				{ID: "song-1", Title: "Song 1", FolderID: "f-album", AlbumID: "alb-rock", TrackNumber: 1},
			})

			rAlbum := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=f-album", nil)
			rAlbum = rAlbum.WithContext(ctx)

			respAlbum, err := api.GetMusicDirectory(rAlbum)
			Expect(err).ToNot(HaveOccurred())
			Expect(respAlbum.Directory).ToNot(BeNil())
			Expect(respAlbum.Directory.CoverArt).To(Equal("al-alb-rock_rockhash"))
		})

		It("resolves library root folder and returns its contents when called with library ID string", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			lib1 := model.Library{ID: 1, Name: "Test Library 1", Path: "/music/library1"}
			rootFolder := &model.Folder{ID: "root-1", LibraryID: 1, Name: ".", Path: ""}
			subFolder := model.Folder{ID: "f-jazz", LibraryID: 1, Name: "Jazz", ParentID: "root-1"}
			folderRepo.SetFolders(*rootFolder, subFolder)
			folderRepo.SetByPath(lib1, ".", rootFolder)

			mockMF := ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
			mockMF.SetData(model.MediaFiles{
				{ID: "loose-1", Title: "Loose Song", FolderID: "root-1", TrackNumber: 1},
			})

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=1", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Directory).ToNot(BeNil())
			Expect(resp.Directory.Id).To(Equal("root-1"))
			Expect(resp.Directory.Name).To(Equal("Test Library 1"))
			Expect(resp.Directory.Child).To(HaveLen(2))
			Expect(resp.Directory.Child[0].Id).To(Equal("f-jazz"))
			Expect(resp.Directory.Child[0].IsDir).To(BeTrue())
			Expect(resp.Directory.Child[1].Id).To(Equal("loose-1"))
			Expect(resp.Directory.Child[1].IsDir).To(BeFalse())
		})

		It("returns artist albums when called with Artist ID (fallback)", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{
				{ID: "art-1", Name: "Pink Floyd"},
			})
			mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
			mockAlbumRepo.SetData(model.Albums{
				{ID: "alb-1", Name: "The Wall", AlbumArtistID: "art-1"},
			})

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=art-1", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Directory).ToNot(BeNil())
			Expect(resp.Directory.Id).To(Equal("art-1"))
			Expect(resp.Directory.Name).To(Equal("Pink Floyd"))
			Expect(resp.Directory.Child).To(HaveLen(1))
			Expect(resp.Directory.Child[0].Id).To(Equal("alb-1"))
			Expect(resp.Directory.Child[0].IsDir).To(BeTrue())
		})

		It("returns album tracks when called with Album ID (fallback)", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
			mockAlbumRepo.SetData(model.Albums{
				{ID: "alb-1", Name: "The Dark Side of the Moon", AlbumArtistID: "art-1"},
			})
			mockMF := ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
			mockMF.SetData(model.MediaFiles{
				{ID: "song-time", Title: "Time", AlbumID: "alb-1", TrackNumber: 4},
			})

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=alb-1", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Directory).ToNot(BeNil())
			Expect(resp.Directory.Id).To(Equal("alb-1"))
			Expect(resp.Directory.Name).To(Equal("The Dark Side of the Moon"))
			Expect(resp.Directory.Child).To(HaveLen(1))
			Expect(resp.Directory.Child[0].Id).To(Equal("song-time"))
			Expect(resp.Directory.Child[0].IsDir).To(BeFalse())
		})

		It("returns ErrorDataNotFound when called with nonexistent ID", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=nonexistent", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
			var subErr subError
			Expect(errors.As(err, &subErr)).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorDataNotFound))
		})

		It("returns ErrorDataNotFound when called with unauthorized library ID", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=99", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
			var subErr subError
			Expect(errors.As(err, &subErr)).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorDataNotFound))
		})

		It("returns ErrorDataNotFound when called with a folder that has Missing: true", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			missingFolder := &model.Folder{ID: "f-missing", LibraryID: 1, Name: "Missing Folder", Missing: true}
			folderRepo.SetFolders(*missingFolder)

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=f-missing", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
			var subErr subError
			Expect(errors.As(err, &subErr)).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorDataNotFound))
		})

		It("returns songs in a folder sorted by disc_number, track_number, title", func() {
			ctx = contextWithUser(ctx, "user-id", 1)
			parentFolder := &model.Folder{ID: "f-multidisc", LibraryID: 1, Name: "Multi Disc Album", ParentID: "root-1"}
			folderRepo.SetFolders(*parentFolder)

			mockMF := ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
			mockMF.SetData(model.MediaFiles{
				{ID: "song-d2-t1", Title: "Disc 2 Track 1", FolderID: "f-multidisc", DiscNumber: 2, TrackNumber: 1},
				{ID: "song-d1-t2", Title: "Disc 1 Track 2", FolderID: "f-multidisc", DiscNumber: 1, TrackNumber: 2},
				{ID: "song-d1-t1", Title: "Disc 1 Track 1", FolderID: "f-multidisc", DiscNumber: 1, TrackNumber: 1},
				{ID: "song-d2-t2", Title: "Disc 2 Track 2", FolderID: "f-multidisc", DiscNumber: 2, TrackNumber: 2},
			})

			r := httptest.NewRequest("GET", "/rest/getMusicDirectory?id=f-multidisc", nil)
			r = r.WithContext(ctx)

			resp, err := api.GetMusicDirectory(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Directory).ToNot(BeNil())
			Expect(resp.Directory.Child).To(HaveLen(4))
			Expect(resp.Directory.Child[0].Id).To(Equal("song-d1-t1"))
			Expect(resp.Directory.Child[1].Id).To(Equal("song-d1-t2"))
			Expect(resp.Directory.Child[2].Id).To(Equal("song-d2-t1"))
			Expect(resp.Directory.Child[3].Id).To(Equal("song-d2-t2"))
		})
	})

	Describe("GetAlbumInfo", func() {
		It("emits image URLs when the album artwork is unresolved", func() {
			api.provider = &fakeInfoProvider{album: &model.Album{ID: "al-1"}}
			r := httptest.NewRequest("GET", "/rest/getAlbumInfo?id=al-1", nil)
			resp, err := api.GetAlbumInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumInfo.SmallImageUrl).ToNot(BeEmpty())
			Expect(resp.AlbumInfo.LargeImageUrl).ToNot(BeEmpty())
		})
		It("omits image URLs when the album artwork is known absent", func() {
			api.provider = &fakeInfoProvider{album: &model.Album{ID: "al-1", ItemImage: model.ItemImage{ImageAbsent: true}}}
			r := httptest.NewRequest("GET", "/rest/getAlbumInfo?id=al-1", nil)
			resp, err := api.GetAlbumInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumInfo.SmallImageUrl).To(BeEmpty())
			Expect(resp.AlbumInfo.MediumImageUrl).To(BeEmpty())
			Expect(resp.AlbumInfo.LargeImageUrl).To(BeEmpty())
		})
	})

	Describe("GetArtistInfo", func() {
		It("emits image URLs when the artist artwork is unresolved", func() {
			api.provider = &fakeInfoProvider{artist: &model.Artist{ID: "ar-1"}}
			r := httptest.NewRequest("GET", "/rest/getArtistInfo?id=ar-1", nil)
			resp, err := api.GetArtistInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ArtistInfo.SmallImageUrl).ToNot(BeEmpty())
		})
		It("omits image URLs when the artist artwork is known absent", func() {
			api.provider = &fakeInfoProvider{artist: &model.Artist{ID: "ar-1", ItemImage: model.ItemImage{ImageAbsent: true}}}
			r := httptest.NewRequest("GET", "/rest/getArtistInfo?id=ar-1", nil)
			resp, err := api.GetArtistInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ArtistInfo.SmallImageUrl).To(BeEmpty())
			Expect(resp.ArtistInfo.MediumImageUrl).To(BeEmpty())
			Expect(resp.ArtistInfo.LargeImageUrl).To(BeEmpty())
		})
	})
})

type fakeInfoProvider struct {
	external.Provider
	album  *model.Album
	artist *model.Artist
}

func (f *fakeInfoProvider) UpdateAlbumInfo(context.Context, string) (*model.Album, error) {
	return f.album, nil
}

func (f *fakeInfoProvider) UpdateArtistInfo(context.Context, string, int, bool) (*model.Artist, error) {
	return f.artist, nil
}
