package artwork

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// pngImage builds a deterministic 2x2 PNG image for a label (color derived from label bytes).
func pngImage(label string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var seed byte
	for i := range len(label) {
		seed += label[i]
	}
	c := color.RGBA{R: seed, G: seed * 3, B: seed * 7, A: 255}
	for y := range 2 {
		for x := range 2 {
			img.Set(x, y, c)
		}
	}
	return img
}

func realPNGBytes(label string) []byte {
	var buf bytes.Buffer
	Expect(png.Encode(&buf, pngImage(label))).To(Succeed())
	return buf.Bytes()
}

var _ = Describe("blurHashUpdater", func() {
	var u *blurHashUpdater
	var ds *tests.MockDataStore
	var repo *tests.MockAlbumRepo
	var version time.Time

	album := func(al model.Album) model.ArtworkID {
		repo = tests.CreateMockAlbumRepo()
		repo.SetData(model.Albums{al})
		ds.MockedAlbum = repo
		return al.CoverArtID()
	}
	stored := func(id string) model.Album {
		al, err := ds.Album(GinkgoT().Context()).Get(id)
		Expect(err).ToNot(HaveOccurred())
		return *al
	}

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		u = newBlurHashUpdater(ds)
		version = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	It("persists a hash computed from the served bytes", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		u.update(GinkgoT().Context(), id, realPNGBytes("x"), version)
		al := stored("al-1")
		Expect(al.BlurHash).ToNot(BeEmpty())
		Expect(al.BlurHashUpdatedAt).To(HaveValue(Equal(version)))
	})

	It("clears the stored hash when the served bytes are a placeholder", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "OLD"})
		u.update(GinkgoT().Context(), id, placeholderImages()[0], version)
		Expect(stored("al-1").BlurHash).To(BeEmpty())
	})

	It("leaves the hash untouched on undecodable bytes", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "KEEP"})
		u.update(GinkgoT().Context(), id, []byte("not an image"), version)
		Expect(stored("al-1").BlurHash).To(Equal("KEEP"))
	})

	It("skips the write when the same bytes are served again under the same version", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		data := realPNGBytes("dedup")
		u.update(GinkgoT().Context(), id, data, version)
		// Tamper with the stored value: a second identical serve must not touch the row.
		Expect(repo.UpdateBlurHash("al-1", "TAMPERED", version)).To(Succeed())
		u.update(GinkgoT().Context(), id, data, version)
		Expect(stored("al-1").BlurHash).To(Equal("TAMPERED"))
	})

	It("re-persists the same hash when the artwork version advances", func() {
		// A scan can bump the entity version without changing the cover; blur_hash_updated_at must
		// follow, or the DTO's staleness gate would emit the fake hash forever after.
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		data := realPNGBytes("same-bytes")
		u.update(GinkgoT().Context(), id, data, version)
		first := stored("al-1")
		newer := version.Add(time.Hour)
		u.update(GinkgoT().Context(), id, data, newer)
		second := stored("al-1")
		Expect(second.BlurHash).To(Equal(first.BlurHash))
		Expect(second.BlurHashUpdatedAt).To(HaveValue(Equal(newer)))
	})

	It("does not write when a placeholder is served and nothing was ever stored", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		u.update(GinkgoT().Context(), id, placeholderImages()[0], version)
		Expect(stored("al-1").BlurHashUpdatedAt).To(BeNil())
	})

	It("ignores non-eligible artwork kinds", func() {
		Expect(func() {
			u.clearIfStored(GinkgoT().Context(), model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: "mf-1"})
		}).ToNot(Panic())
		Expect(u.seen).To(BeEmpty())
	})

	It("dedups across artwork ids that differ only in their embedded timestamp", func() {
		// Client coverArt tokens embed a LastUpdate; the seen key must ignore it, or every scan bump
		// would defeat the dedup and re-decode identical bytes.
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		data := realPNGBytes("dedup")
		u.update(GinkgoT().Context(), id, data, version)
		Expect(repo.UpdateBlurHash("al-1", "TAMPERED", version)).To(Succeed())
		bumped := id
		bumped.LastUpdate = version.Add(time.Hour)
		u.update(GinkgoT().Context(), bumped, data, version)
		Expect(stored("al-1").BlurHash).To(Equal("TAMPERED"))
		Expect(u.seen).To(HaveLen(1))
	})
})
