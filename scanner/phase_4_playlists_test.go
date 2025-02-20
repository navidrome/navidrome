package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("phasePlaylists", func() {
	var (
		phase      *phasePlaylists
		ctx        context.Context
		state      *scanState
		folderRepo *mockFolderRepository
		ds         *tests.MockDataStore
		pls        *mockPlaylists
		cw         artwork.CacheWarmer
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.AutoImportPlaylists = true
		ctx = context.Background()
		ctx = request.WithUser(ctx, model.User{ID: "123", IsAdmin: true})
		folderRepo = &mockFolderRepository{}
		ds = &tests.MockDataStore{
			MockedFolder: folderRepo,
		}
		pls = &mockPlaylists{}
		cw = artwork.NoopCacheWarmer()
		state = &scanState{}
		phase = createPhasePlaylists(ctx, state, ds, pls, cw)
	})

	Describe("description", func() {
		It("returns the correct description", func() {
			Expect(phase.description()).To(Equal("Import/update playlists"))
		})
	})

	Describe("producer", func() {
		It("produces folders with playlists", func() {
			folderRepo.SetData(map[*model.Folder]error{
				{Path: "/path/to/folder1"}: nil,
				{Path: "/path/to/folder2"}: nil,
			})

			var produced []*model.Folder
			err := phase.produce(func(folder *model.Folder) {
				produced = append(produced, folder)
			})

			sort.Slice(produced, func(i, j int) bool {
				return produced[i].Path < produced[j].Path
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(produced).To(HaveLen(2))
			Expect(produced[0].Path).To(Equal("/path/to/folder1"))
			Expect(produced[1].Path).To(Equal("/path/to/folder2"))
		})

		It("returns an error if there is an error loading folders", func() {
			folderRepo.SetData(map[*model.Folder]error{
				nil: errors.New("error loading folders"),
			})

			called := false
			err := phase.produce(func(folder *model.Folder) { called = true })

			Expect(err).To(HaveOccurred())
			Expect(called).To(BeFalse())
			Expect(err).To(MatchError(ContainSubstring("error loading folders")))
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

			pls.On("ImportFile", mock.Anything, folder, "playlist1.m3u").
				Return(&model.Playlist{}, nil)
			pls.On("ImportFile", mock.Anything, folder, "playlist2.m3u").
				Return(&model.Playlist{}, nil)

			_, err := phase.processPlaylistsInFolder(folder)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Calls).To(HaveLen(2))
			Expect(pls.Calls[0].Arguments[2]).To(Equal("playlist1.m3u"))
			Expect(pls.Calls[1].Arguments[2]).To(Equal("playlist2.m3u"))
			Expect(phase.refreshed.Load()).To(Equal(uint32(2)))
		})

		It("reports an error if there is an error reading files", func() {
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
	core.Playlists
}

func (p *mockPlaylists) ImportFile(ctx context.Context, folder *model.Folder, filename string) (*model.Playlist, error) {
	args := p.Called(ctx, folder, filename)
	return args.Get(0).(*model.Playlist), args.Error(1)
}

type mockFolderRepository struct {
	model.FolderRepository
	data map[*model.Folder]error
}

func (f *mockFolderRepository) GetTouchedWithPlaylists() (model.FolderCursor, error) {
	return func(yield func(model.Folder, error) bool) {
		for folder, err := range f.data {
			if err != nil {
				if !yield(model.Folder{}, err) {
					return
				}
				continue
			}
			if !yield(*folder, err) {
				return
			}
		}
	}, nil
}

func (f *mockFolderRepository) SetData(m map[*model.Folder]error) {
	f.data = m
}
