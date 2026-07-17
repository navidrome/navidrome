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
			a:       &artwork{ds: ds},
			buffer:  make(map[model.ArtworkID]enqueueRequest),
			wake:    make(chan struct{}, 1),
			started: true,
		}
	})

	Describe("Enqueue", func() {
		It("accepts album, artist and playlist artwork and dedups, keeping the newest snapshot", func() {
			id := model.Album{ID: "al-1"}.CoverArtID()
			t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			t2 := t1.Add(time.Hour)
			u.Enqueue(id, t1)
			u.Enqueue(id, t2)
			u.Enqueue(model.Artist{ID: "ar-1"}.CoverArtID(), t1)
			Expect(u.buffer).To(HaveLen(2))
			Expect(u.buffer[id].snapshot).To(Equal(t2))
			Expect(u.buffer[id].gone).To(BeFalse())
		})

		It("merges a gone flag onto a pending snapshot for the same artwork", func() {
			id := model.Album{ID: "al-1"}.CoverArtID()
			t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			u.Enqueue(id, t1)
			u.EnqueueGone(id)
			Expect(u.buffer[id].snapshot).To(Equal(t1))
			Expect(u.buffer[id].gone).To(BeTrue())
		})

		It("ignores other artwork kinds", func() {
			u.Enqueue(model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: "mf-1"}, time.Time{})
			u.EnqueueGone(model.ArtworkID{Kind: model.KindRadioArtwork, ID: "ra-1"})
			Expect(u.buffer).To(BeEmpty())
		})
	})

	Describe("process", func() {
		var version time.Time

		BeforeEach(func() {
			version = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		})

		It("skips entities whose stored hash is at or after the snapshot", func() {
			al := model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "LEHV6nWB2yk8", BlurHashUpdatedAt: &version}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo

			u.process(GinkgoT().Context(), al.CoverArtID(), enqueueRequest{snapshot: version})
			stored, err := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.BlurHash).To(Equal("LEHV6nWB2yk8"))
		})

		It("keeps the stored hash when a recompute fails transiently", func() {
			al := model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "LEHV6nWB2yk8", BlurHashUpdatedAt: nil}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo
			ds.MockedFolder = failingFolderRepo{}

			// A newer snapshot forces a recompute, but the reader chain errors (transient): the stored
			// hash must survive, so clients don't churn on a fake until a later fill succeeds.
			newer := version.Add(time.Hour)
			u.process(GinkgoT().Context(), al.CoverArtID(), enqueueRequest{snapshot: newer})
			stored, err := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.BlurHash).To(Equal("LEHV6nWB2yk8"))
		})

		It("clears a stored hash when the source is gone", func() {
			al := model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "LEHV6nWB2yk8", BlurHashUpdatedAt: &version}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo

			u.process(GinkgoT().Context(), al.CoverArtID(), enqueueRequest{gone: true})
			stored, err := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.BlurHash).To(BeEmpty())
		})

		It("does nothing for a gone serve with no stored hash", func() {
			al := model.Album{ID: "al-1", UpdatedAt: version}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo

			Expect(func() {
				u.process(GinkgoT().Context(), al.CoverArtID(), enqueueRequest{gone: true})
			}).ToNot(Panic())
			stored, _ := ds.Album(GinkgoT().Context()).Get("al-1")
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
