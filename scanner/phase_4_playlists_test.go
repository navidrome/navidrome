package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("phasePlaylists", func() {
	var (
		phase *phasePlaylists
		ctx   context.Context
		state *scanState
		ds    model.DataStore
		pls   *mockPlaylists
	)

	putPlaylistFolders := func(paths ...string) {
		GinkgoHelper()
		lib, err := ds.Library().Get(ctx, model.DefaultLibraryID)
		Expect(err).ToNot(HaveOccurred())
		for _, path := range paths {
			f := model.NewFolder(*lib, path)
			f.NumPlaylists = 1
			Expect(ds.Folder().Put(ctx, f)).To(Succeed())
		}
	}
	removeAdmin := func() {
		GinkgoHelper()
		Expect(ds.User().Delete(request.WithUser(ctx, model.User{IsAdmin: true}), "123")).To(Succeed())
	}
	withStub := func(wrap func(model.DataStore) model.DataStore) {
		phase = createPhasePlaylists(ctx, state, persistencetest.Overlay(ds, wrap), pls)
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		conf.Update(func(c *conf.Options) { c.AutoImportPlaylists = true })
		ds = persistence.New(db.Db())
		// An admin user exists by default, so playlist import proceeds.
		Expect(ds.User().Put(ctx, &model.User{ID: "123", UserName: "admin", IsAdmin: true})).To(Succeed())
		pls = &mockPlaylists{}
		state = &scanState{}
		phase = createPhasePlaylists(ctx, state, ds, pls)
	})

	Describe("description", func() {
		It("returns the correct description", func() {
			Expect(phase.description()).To(Equal("Import/update playlists"))
		})
	})

	Describe("producer", func() {
		It("produces folders with playlists", func() {
			putPlaylistFolders("path/to/folder1", "path/to/folder2")
			lib, err := ds.Library().Get(ctx, model.DefaultLibraryID)
			Expect(err).ToNot(HaveOccurred())
			Expect(ds.Folder().Put(ctx, model.NewFolder(*lib, "path/to/no-playlists"))).To(Succeed())

			var produced []*model.Folder
			err = phase.produce(func(folder *model.Folder) {
				produced = append(produced, folder)
			})

			sort.Slice(produced, func(i, j int) bool {
				return produced[i].Name < produced[j].Name
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(produced).To(HaveLen(2))
			Expect(produced[0].Path).To(Equal("path/to"))
			Expect(produced[0].Name).To(Equal("folder1"))
			Expect(produced[1].Name).To(Equal("folder2"))
		})

		It("returns an error if there is an error loading folders", func() {
			putPlaylistFolders("path/to/folder1")
			withStub(func(ds model.DataStore) model.DataStore {
				return &failingFolderDS{DataStore: ds, err: errors.New("error loading folders")}
			})

			called := false
			err := phase.produce(func(folder *model.Folder) { called = true })

			Expect(err).To(HaveOccurred())
			Expect(called).To(BeFalse())
			Expect(err).To(MatchError(ContainSubstring("error loading folders")))
		})

		It("sets the pending flag and imports nothing when no admin user exists", func() {
			// Remove the admin user; produce resolves the admin at phase time.
			removeAdmin()
			putPlaylistFolders("path/to/folder1")

			called := false
			err := phase.produce(func(folder *model.Folder) { called = true })

			Expect(err).ToNot(HaveOccurred())
			Expect(called).To(BeFalse())
			v, _ := ds.Property().Get(ctx, consts.PlaylistsImportPendingFlagKey)
			Expect(v).To(Equal("1"))
		})

		It("returns an error (not a silent defer) on a datastore failure resolving the admin", func() {
			withStub(func(ds model.DataStore) model.DataStore {
				return &failingAdminDS{DataStore: ds, err: errors.New("db is locked")}
			})

			err := phase.produce(func(folder *model.Folder) {})

			Expect(err).To(MatchError(ContainSubstring("finding admin user")))
			// Must NOT have set the pending flag on a real error.
			_, getErr := ds.Property().Get(ctx, consts.PlaylistsImportPendingFlagKey)
			Expect(getErr).To(HaveOccurred())
		})

		It("returns an error when the pending flag cannot be persisted", func() {
			removeAdmin() // no admin -> defer path
			withStub(func(ds model.DataStore) model.DataStore {
				return &failingPropertyDS{DataStore: ds, err: errors.New("property table unavailable")}
			})

			err := phase.produce(func(folder *model.Folder) {})

			Expect(err).To(MatchError(ContainSubstring("recording pending playlist import")))
		})

		It("imports all playlist folders when the pending flag is set", func() {
			Expect(ds.Property().Put(ctx, consts.PlaylistsImportPendingFlagKey, "1")).To(Succeed())
			putPlaylistFolders("path/to/folder1", "path/to/folder2")
			Expect(ds.Library().ScanEnd(ctx, model.DefaultLibraryID)).To(Succeed())
			// Touched set is empty: proves selection used GetAllWithPlaylists.
			touched, err := ds.Folder().GetTouchedWithPlaylists(ctx)
			Expect(err).ToNot(HaveOccurred())
			count := 0
			for range touched {
				count++
			}
			Expect(count).To(BeZero())

			var produced []*model.Folder
			err = phase.produce(func(folder *model.Folder) { produced = append(produced, folder) })

			Expect(err).ToNot(HaveOccurred())
			Expect(produced).To(HaveLen(2))
			Expect(phase.pendingImport).To(BeTrue())
		})
	})

	Describe("finalize", func() {
		It("clears the pending flag after a successful pending import", func() {
			Expect(ds.Property().Put(ctx, consts.PlaylistsImportPendingFlagKey, "1")).To(Succeed())
			phase.pendingImport = true

			Expect(phase.finalize(nil)).To(Succeed())

			_, err := ds.Property().Get(ctx, consts.PlaylistsImportPendingFlagKey)
			Expect(err).To(HaveOccurred()) // deleted
		})

		It("keeps the pending flag when the import failed", func() {
			Expect(ds.Property().Put(ctx, consts.PlaylistsImportPendingFlagKey, "1")).To(Succeed())
			phase.pendingImport = true

			Expect(phase.finalize(errors.New("boom"))).ToNot(Succeed())

			v, _ := ds.Property().Get(ctx, consts.PlaylistsImportPendingFlagKey)
			Expect(v).To(Equal("1"))
		})
	})

	Describe("processPlaylistsInFolder", func() {
		It("processes playlists in a folder", func() {
			libPath := GinkgoT().TempDir()
			folder := &model.Folder{LibraryPath: libPath, Path: "path/to", Name: "folder"}
			_ = os.MkdirAll(folder.AbsolutePath(), 0755)

			file1 := filepath.Join(folder.AbsolutePath(), "playlist1.m3u")
			file2 := filepath.Join(folder.AbsolutePath(), "playlist2.m3u")
			_ = os.WriteFile(file1, []byte{}, 0600)
			_ = os.WriteFile(file2, []byte{}, 0600)

			pls.On("ImportFromFolder", mock.Anything, folder, "playlist1.m3u").
				Return(&model.Playlist{}, nil)
			pls.On("ImportFromFolder", mock.Anything, folder, "playlist2.m3u").
				Return(&model.Playlist{}, nil)

			_, err := phase.processPlaylistsInFolder(folder)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Calls).To(HaveLen(2))
			Expect(pls.Calls[0].Arguments[2]).To(Equal("playlist1.m3u"))
			Expect(pls.Calls[1].Arguments[2]).To(Equal("playlist2.m3u"))
			Expect(phase.refreshed.Load()).To(Equal(uint32(2)))
		})

		It("enqueues artwork resolution for the imported playlist", func() {
			libPath := GinkgoT().TempDir()
			folder := &model.Folder{LibraryPath: libPath, Path: "path/to", Name: "folder"}
			_ = os.MkdirAll(folder.AbsolutePath(), 0755)

			file1 := filepath.Join(folder.AbsolutePath(), "playlist1.m3u")
			_ = os.WriteFile(file1, []byte{}, 0600)

			pls.On("ImportFromFolder", mock.Anything, folder, "playlist1.m3u").
				Return(&model.Playlist{ID: "pl1"}, nil)

			_, err := phase.processPlaylistsInFolder(folder)
			Expect(err).ToNot(HaveOccurred())

			queued, err := ds.ArtworkQueue().DequeueBatch(ctx, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(queued).To(ContainElement(SatisfyAll(
				HaveField("ItemKind", "pl"),
				HaveField("ItemID", "pl1"),
				HaveField("Priority", model.ArtworkPriorityScan),
			)))
		})

		It("reports an error if there is an error reading files", func() {
			tests.SkipOnWindows("relies on Unix /etc filesystem")
			progress := make(chan *ProgressInfo)
			state.progress = progress
			folder := &model.Folder{Path: "/invalid/path"}
			go func() {
				_, err := phase.processPlaylistsInFolder(folder)
				// I/O errors are ignored
				Expect(err).ToNot(HaveOccurred())
			}()

			// But are reported
			info := &ProgressInfo{}
			Eventually(progress).Should(Receive(&info))
			Expect(info.Warning).To(ContainSubstring("no such file or directory"))
		})
	})
})

type mockPlaylists struct {
	mock.Mock
	playlists.Playlists
}

func (p *mockPlaylists) ImportFromFolder(ctx context.Context, folder *model.Folder, filename string) (*model.Playlist, error) {
	args := p.Called(ctx, folder, filename)
	return args.Get(0).(*model.Playlist), args.Error(1)
}

type failingFolderDS struct {
	model.DataStore
	err error
}

func (d *failingFolderDS) Folder() model.FolderRepository {
	return &failingFolderRepo{FolderRepository: d.DataStore.Folder(), err: d.err}
}

// failingFolderRepo's cursor fails on its first row.
type failingFolderRepo struct {
	model.FolderRepository
	err error
}

func (f *failingFolderRepo) GetTouchedWithPlaylists(context.Context) (model.FolderCursor, error) {
	return func(yield func(model.Folder, error) bool) { yield(model.Folder{}, f.err) }, nil
}

type failingAdminDS struct {
	model.DataStore
	err error
}

func (d *failingAdminDS) User() model.UserRepository {
	return &failingAdminRepo{UserRepository: d.DataStore.User(), err: d.err}
}

type failingAdminRepo struct {
	model.UserRepository
	err error
}

func (r *failingAdminRepo) FindFirstAdmin(context.Context) (*model.User, error) {
	return nil, r.err
}

type failingPropertyDS struct {
	model.DataStore
	err error
}

func (d *failingPropertyDS) Property() model.PropertyRepository {
	return &failingPropertyRepo{PropertyRepository: d.DataStore.Property(), err: d.err}
}

type failingPropertyRepo struct {
	model.PropertyRepository
	err error
}

func (r *failingPropertyRepo) Put(context.Context, string, string) error {
	return r.err
}
