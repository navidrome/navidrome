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
			a:        &artwork{ds: ds},
			buffer:   make(map[model.ArtworkID]bool),
			noResult: make(map[model.ArtworkID]time.Time),
			wake:     make(chan struct{}, 1),
		}
	})

	Describe("Enqueue", func() {
		It("accepts album, artist and playlist artwork and dedups, keeping the force flag sticky", func() {
			id := model.Album{ID: "al-1"}.CoverArtID()
			u.Enqueue(id, true)
			u.Enqueue(id, false)
			u.Enqueue(model.Artist{ID: "ar-1"}.CoverArtID(), false)
			Expect(u.buffer).To(HaveLen(2))
			Expect(u.buffer[id]).To(BeTrue())
		})

		It("ignores other artwork kinds", func() {
			u.Enqueue(model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: "mf-1"}, false)
			u.Enqueue(model.ArtworkID{Kind: model.KindRadioArtwork, ID: "ra-1"}, true)
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

			u.process(GinkgoT().Context(), al.CoverArtID(), false)
			stored, err := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.BlurHash).To(Equal("LEHV6nWB2yk8"))
		})

		It("skips entities that previously yielded no result for the same artwork version", func() {
			al := model.Album{ID: "al-1", UpdatedAt: version}
			repo := tests.CreateMockAlbumRepo()
			repo.SetData(model.Albums{al})
			ds.MockedAlbum = repo
			u.setNoResult(al.CoverArtID(), al.ArtworkUpdatedAt())

			u.process(GinkgoT().Context(), al.CoverArtID(), false)
			stored, _ := ds.Album(GinkgoT().Context()).Get("al-1")
			Expect(stored.BlurHash).To(BeEmpty())
		})

		It("does nothing when the entity is gone", func() {
			ds.MockedAlbum = tests.CreateMockAlbumRepo()
			Expect(func() { u.process(GinkgoT().Context(), model.Album{ID: "missing"}.CoverArtID(), false) }).ToNot(Panic())
		})
	})
})
