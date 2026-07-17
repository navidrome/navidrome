package artwork

import (
	"errors"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// failingFolderRepo makes the artwork reader chain fail with a clean (transient-style) error.
type failingFolderRepo struct{ model.FolderRepository }

func (failingFolderRepo) GetAll(...model.QueryOptions) ([]model.Folder, error) {
	return nil, errors.New("boom")
}

var _ = Describe("blurHashUpdater", func() {
	var u *blurHashUpdater
	var ds *tests.MockDataStore

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		// started is pre-set so Enqueue never spawns run(): tests drive next()/process() directly.
		u = &blurHashUpdater{
			a:        &artwork{ds: ds},
			buffer:   make(map[model.ArtworkID]enqueueRequest),
			noResult: make(map[model.ArtworkID]noResultEntry),
			wake:     make(chan struct{}, 1),
			started:  true,
		}
	})

	Describe("Enqueue", func() {
		It("accepts album, artist and playlist artwork and dedups, merging force and newest image time", func() {
			id := model.Album{ID: "al-1"}.CoverArtID()
			t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			t2 := t1.Add(time.Hour)
			u.Enqueue(id, t2, true)
			u.Enqueue(id, t1, false)
			u.Enqueue(model.Artist{ID: "ar-1"}.CoverArtID(), t1, false)
			Expect(u.buffer).To(HaveLen(2))
			Expect(u.buffer[id].force).To(BeTrue())
			Expect(u.buffer[id].imageUpdatedAt).To(Equal(t2))
		})

		It("ignores other artwork kinds", func() {
			u.Enqueue(model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: "mf-1"}, time.Time{}, false)
			u.Enqueue(model.ArtworkID{Kind: model.KindRadioArtwork, ID: "ra-1"}, time.Time{}, true)
			Expect(u.buffer).To(BeEmpty())
		})
	})

	Describe("process", func() {
		var version time.Time

		BeforeEach(func() {
			version = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		})

		It("skips entities whose stored hash matches the current artwork version", func() {
			al := model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "LEHV6nWB2yk8", BlurHashUpdatedAt: &version}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo

			u.process(GinkgoT().Context(), al.CoverArtID(), enqueueRequest{imageUpdatedAt: version})
			stored, err := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.BlurHash).To(Equal("LEHV6nWB2yk8"))
		})

		It("skips entities that previously yielded no result for the same signals", func() {
			al := model.Album{ID: "al-1", UpdatedAt: version}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo
			u.setNoResult(al.CoverArtID(), version)

			u.process(GinkgoT().Context(), al.CoverArtID(), enqueueRequest{imageUpdatedAt: version})
			stored, _ := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(stored.BlurHash).To(BeEmpty())
		})

		It("memoizes a failed retry under the newer signal", func() {
			al := model.Album{ID: "al-1", UpdatedAt: version}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo
			ds.MockedFolder = failingFolderRepo{}
			u.setNoResult(al.CoverArtID(), version)

			// A newer image mtime bypasses the no-result skip; the compute fails cleanly here, so
			// the entry is refreshed under the newer signal, with the TTL as the retry bound.
			newer := version.Add(time.Hour)
			u.process(GinkgoT().Context(), al.CoverArtID(), enqueueRequest{imageUpdatedAt: newer})
			last, ok := u.lastNoResult(al.CoverArtID())
			Expect(ok).To(BeTrue())
			Expect(last.sig).To(Equal(newer))
		})

		It("clears a stored hash when a recompute with change evidence yields no result", func() {
			storedAt := version.Add(-time.Hour)
			al := model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "LEHV6nWB2yk8", BlurHashUpdatedAt: &storedAt}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo
			ds.MockedFolder = failingFolderRepo{}

			// storedAt < version = change evidence; the compute fails, so the stale hash must go.
			u.process(GinkgoT().Context(), al.CoverArtID(), enqueueRequest{imageUpdatedAt: version})
			stored, err := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.BlurHash).To(BeEmpty())
		})

		It("does nothing when the entity is gone", func() {
			ds.MockedAlbum = tests.CreateMockAlbumRepo()
			Expect(func() {
				u.process(GinkgoT().Context(), model.Album{ID: "missing"}.CoverArtID(), enqueueRequest{})
			}).ToNot(Panic())
		})
	})
})
