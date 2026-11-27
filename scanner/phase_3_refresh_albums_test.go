package scanner

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("phaseRefreshAlbums", func() {
	var (
		phase *phaseRefreshAlbums
		ctx   context.Context
		ds    model.DataStore
		libs  model.Libraries
		state *scanState
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		Expect(ds.Library().Put(ctx, &model.Library{ID: 2, Name: "Library 2", Path: "/music2"})).To(Succeed())
		libs = model.Libraries{
			{ID: 1, Name: "Library 1"},
			{ID: 2, Name: "Library 2"},
		}
		state = &scanState{libraries: libs}
		phase = createPhaseRefreshAlbums(ctx, state, ds)
	})

	Describe("description", func() {
		It("returns the correct description", func() {
			Expect(phase.description()).To(Equal("Refresh all new/changed albums"))
		})
	})

	Describe("producer", func() {
		It("produces albums that need refreshing", func() {
			Expect(ds.Album().Put(ctx, &model.Album{LibraryID: 2, ID: "album2", Name: "Album 2"})).To(Succeed())
			Expect(ds.Library().ScanEnd(ctx, 2)).To(Succeed())
			Expect(ds.Album().Put(ctx, &model.Album{LibraryID: 1, ID: "album1", Name: "Album 1"})).To(Succeed())

			var produced []*model.Album
			err := phase.produce(func(album *model.Album) {
				produced = append(produced, album)
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(produced).To(HaveLen(1))
			Expect(produced[0].ID).To(Equal("album1"))
		})

		It("returns an error if there is an error loading albums", func() {
			phase = createPhaseRefreshAlbums(ctx, state, withFailingAlbums(ds, errors.New("db error")))

			err := phase.produce(func(album *model.Album) {})

			Expect(err).To(MatchError(ContainSubstring("loading touched albums")))
		})
	})

	Describe("filterUnmodified", func() {
		It("filters out unmodified albums", func() {
			album := &model.Album{ID: "album1", Name: "Album 1", SongCount: 1, LibraryID: 1,
				FolderIDs: []string{"folder1"}, Discs: model.Discs{1: ""}}
			Expect(ds.MediaFile().Put(ctx, &model.MediaFile{ID: "mf1", LibraryID: 1, AlbumID: "album1",
				Title: "Song 1", Album: "Album 1", FolderID: "folder1"})).To(Succeed())

			result, err := phase.filterUnmodified(album)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
		It("keep modified albums", func() {
			album := &model.Album{ID: "album1", Name: "Album 1"}
			Expect(ds.MediaFile().Put(ctx, &model.MediaFile{ID: "mf1", LibraryID: 1, AlbumID: "album1",
				Title: "Song 1", Album: "Album 2"})).To(Succeed())

			result, err := phase.filterUnmodified(album)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.ID).To(Equal("album1"))
		})
		It("skips albums with no media files", func() {
			album := &model.Album{ID: "album1", Name: "Album 1"}

			result, err := phase.filterUnmodified(album)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("refreshAlbum", func() {
		It("refreshes the album in the database", func() {
			Expect(ds.Album().CountAll(ctx)).To(Equal(int64(0)))

			album := &model.Album{ID: "album1", Name: "Album 1", LibraryID: 1}
			result, err := phase.refreshAlbum(album)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.ID).To(Equal("album1"))

			savedAlbum, err := ds.Album().Get(ctx, "album1")
			Expect(err).ToNot(HaveOccurred())

			Expect(savedAlbum).ToNot(BeNil())
			Expect(savedAlbum.Name).To(Equal("Album 1"))
			Expect(phase.refreshed.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())
		})

		It("returns an error if there is an error refreshing the album", func() {
			phase = createPhaseRefreshAlbums(ctx, state, withFailingAlbums(ds, errors.New("db error")))
			album := &model.Album{ID: "album1", Name: "Album 1", LibraryID: 1}

			result, err := phase.refreshAlbum(album)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring("refreshing album")))
			Expect(phase.refreshed.Load()).To(Equal(uint32(0)))
			Expect(state.changesDetected.Load()).To(BeFalse())
		})
	})
})

// withFailingAlbums fails reading touched albums and saving albums.
func withFailingAlbums(ds model.DataStore, err error) model.DataStore {
	return persistencetest.Stub(ds, persistencetest.Repos{Album: func(r model.AlbumRepository) model.AlbumRepository {
		return &failingAlbumRepo{AlbumRepository: r, err: err}
	}})
}

type failingAlbumRepo struct {
	model.AlbumRepository
	err error
}

func (r *failingAlbumRepo) GetTouchedAlbums(context.Context, int) (model.AlbumCursor, error) {
	return nil, r.err
}

func (r *failingAlbumRepo) Put(context.Context, *model.Album) error {
	return r.err
}
