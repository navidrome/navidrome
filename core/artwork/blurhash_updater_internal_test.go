package artwork

import (
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("blurHashUpdater", func() {
	var u *blurHashUpdater
	var ds *tests.MockDataStore

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		// No run() goroutine: tests drive next()/process() directly.
		u = &blurHashUpdater{
			a:      &artwork{ds: ds},
			buffer: make(map[model.ArtworkID]struct{}),
			wake:   make(chan struct{}, 1),
		}
	})

	Describe("Enqueue", func() {
		It("accepts album, artist and playlist artwork and dedups", func() {
			id := model.Album{ID: "al-1"}.CoverArtID()
			u.Enqueue(id)
			u.Enqueue(id)
			u.Enqueue(model.Artist{ID: "ar-1"}.CoverArtID())
			Expect(u.buffer).To(HaveLen(2))
		})

		It("ignores other artwork kinds", func() {
			u.Enqueue(model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: "mf-1"})
			u.Enqueue(model.ArtworkID{Kind: model.KindRadioArtwork, ID: "ra-1"})
			Expect(u.buffer).To(BeEmpty())
		})
	})

	Describe("process", func() {
		It("skips entities whose stored hash matches the current artwork version", func() {
			version := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			al := model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "LEHV6nWB2yk8", BlurHashUpdatedAt: &version}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo

			// u.a has no cache: if process tried to compute, it would panic. Not panicking proves the skip.
			Expect(func() { u.process(GinkgoT().Context(), al.CoverArtID()) }).ToNot(Panic())
			stored, err := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.BlurHash).To(Equal("LEHV6nWB2yk8"))
		})

		It("does nothing when the entity is gone", func() {
			ds.MockedAlbum = tests.CreateMockAlbumRepo()
			Expect(func() { u.process(GinkgoT().Context(), model.Album{ID: "missing"}.CoverArtID()) }).ToNot(Panic())
		})
	})
})
