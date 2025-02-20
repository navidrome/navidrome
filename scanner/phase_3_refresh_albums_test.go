package scanner

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("phaseRefreshAlbums", func() {
	var (
		phase     *phaseRefreshAlbums
		ctx       context.Context
		albumRepo *tests.MockAlbumRepo
		mfRepo    *tests.MockMediaFileRepo
		ds        *tests.MockDataStore
		libs      model.Libraries
		state     *scanState
	)

	BeforeEach(func() {
		ctx = context.Background()
		albumRepo = tests.CreateMockAlbumRepo()
		mfRepo = tests.CreateMockMediaFileRepo()
		ds = &tests.MockDataStore{
			MockedAlbum:     albumRepo,
			MockedMediaFile: mfRepo,
		}
		libs = model.Libraries{
			{ID: 1, Name: "Library 1"},
			{ID: 2, Name: "Library 2"},
		}
		state = &scanState{}
		phase = createPhaseRefreshAlbums(ctx, state, ds, libs)
	})

	Describe("description", func() {
		It("returns the correct description", func() {
			Expect(phase.description()).To(Equal("Refresh all new/changed albums"))
		})
	})

	Describe("producer", func() {
		It("produces albums that need refreshing", func() {
			albumRepo.SetData(model.Albums{
				{LibraryID: 1, ID: "album1", Name: "Album 1"},
			})

			var produced []*model.Album
			err := phase.produce(func(album *model.Album) {
				produced = append(produced, album)
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(produced).To(HaveLen(1))
			Expect(produced[0].ID).To(Equal("album1"))
		})

		It("returns an error if there is an error loading albums", func() {
			albumRepo.SetData(model.Albums{
				{ID: "error"},
			})

			err := phase.produce(func(album *model.Album) {})

			Expect(err).To(MatchError(ContainSubstring("loading touched albums")))
		})
	})

	Describe("filterUnmodified", func() {
		It("filters out unmodified albums", func() {
			album := &model.Album{ID: "album1", Name: "Album 1", SongCount: 1,
				FolderIDs: []string{"folder1"}, Discs: model.Discs{1: ""}}
			mfRepo.SetData(model.MediaFiles{
				{AlbumID: "album1", Title: "Song 1", Album: "Album 1", FolderID: "folder1"},
			})

			result, err := phase.filterUnmodified(album)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
		It("keep modified albums", func() {
			album := &model.Album{ID: "album1", Name: "Album 1"}
			mfRepo.SetData(model.MediaFiles{
				{AlbumID: "album1", Title: "Song 1", Album: "Album 2"},
			})

			result, err := phase.filterUnmodified(album)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.ID).To(Equal("album1"))
		})
		It("skips albums with no media files", func() {
			album := &model.Album{ID: "album1", Name: "Album 1"}
			mfRepo.SetData(model.MediaFiles{})

			result, err := phase.filterUnmodified(album)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("refreshAlbum", func() {
		It("refreshes the album in the database", func() {
			Expect(albumRepo.CountAll()).To(Equal(int64(0)))

			album := &model.Album{ID: "album1", Name: "Album 1"}
			result, err := phase.refreshAlbum(album)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.ID).To(Equal("album1"))

			savedAlbum, err := albumRepo.Get("album1")
			Expect(err).ToNot(HaveOccurred())

			Expect(savedAlbum).ToNot(BeNil())
			Expect(savedAlbum.ID).To(Equal("album1"))
			Expect(phase.refreshed.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())
		})

		It("returns an error if there is an error refreshing the album", func() {
			album := &model.Album{ID: "album1", Name: "Album 1"}
			albumRepo.SetError(true)

			result, err := phase.refreshAlbum(album)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring("refreshing album")))
			Expect(phase.refreshed.Load()).To(Equal(uint32(0)))
			Expect(state.changesDetected.Load()).To(BeFalse())
		})
	})
})
