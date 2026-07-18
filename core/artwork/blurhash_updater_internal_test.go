package artwork

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// hugePNGHeader builds a valid PNG signature+IHDR declaring a 50000x50000 raster with no pixel data:
// enough for DecodeConfig to report the dimensions the decode gate must reject.
func hugePNGHeader() []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], 50000)
	binary.BigEndian.PutUint32(ihdr[4:], 50000)
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // RGBA
	var chunk bytes.Buffer
	chunk.WriteString("IHDR")
	chunk.Write(ihdr)
	_ = binary.Write(&buf, binary.BigEndian, uint32(13))
	buf.Write(chunk.Bytes())
	_ = binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(chunk.Bytes()))
	return buf.Bytes()
}

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
		u.update(GinkgoT().Context(), id, realPNGBytes("x"), version, time.Now())
		al := stored("al-1")
		Expect(al.BlurHash).ToNot(BeEmpty())
		Expect(al.BlurHashUpdatedAt).To(HaveValue(Equal(version)))
	})

	It("clears the stored hash when the served bytes are a placeholder", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "OLD"})
		u.update(GinkgoT().Context(), id, placeholderImages()[0], version, time.Now())
		Expect(stored("al-1").BlurHash).To(BeEmpty())
	})

	It("leaves the hash untouched on undecodable bytes", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "KEEP"})
		u.update(GinkgoT().Context(), id, []byte("not an image"), version, time.Now())
		Expect(stored("al-1").BlurHash).To(Equal("KEEP"))
	})

	It("does not rewrite when the stored hash is current for the entity version", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		data := realPNGBytes("dedup")
		u.update(GinkgoT().Context(), id, data, version, time.Now())
		first := stored("al-1")
		// A later serve of the same bytes (newer tee version, unchanged entity) must not move the row.
		u.update(GinkgoT().Context(), id, data, version.Add(time.Hour), time.Now())
		Expect(stored("al-1").BlurHashUpdatedAt).To(HaveValue(Equal(*first.BlurHashUpdatedAt)))
	})

	It("clamps the persisted version up to the entity's artwork version", func() {
		// The read-side version may over-approximate (folder parents); after a serve the hash is fresh
		// by construction, so the write clamps up and the DTO accepts it — omission windows close.
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		data := realPNGBytes("clamp")
		u.update(GinkgoT().Context(), id, data, version, time.Now())
		first := stored("al-1")

		newer := version.Add(time.Hour)
		album(model.Album{ID: "al-1", UpdatedAt: newer, BlurHash: first.BlurHash, BlurHashUpdatedAt: first.BlurHashUpdatedAt})
		u.update(GinkgoT().Context(), id, data, version, time.Now()) // same bytes, old tee version
		second := stored("al-1")
		Expect(second.BlurHash).To(Equal(first.BlurHash))
		Expect(second.BlurHashUpdatedAt).To(HaveValue(Equal(newer)))
	})

	It("restores the stored hash when it drifts from the served bytes", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		data := realPNGBytes("truth")
		u.update(GinkgoT().Context(), id, data, version, time.Now())
		truth := stored("al-1").BlurHash
		Expect(repo.UpdateBlurHash("al-1", "DRIFTED", version)).To(Succeed())
		u.update(GinkgoT().Context(), id, data, version, time.Now())
		Expect(stored("al-1").BlurHash).To(Equal(truth))
	})

	It("skips images whose declared dimensions exceed the decode bound", func() {
		// The tee's byte cap limits compressed size only; a decompression bomb must be rejected from
		// the header before the raster is allocated.
		id := album(model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "KEEP"})
		u.update(GinkgoT().Context(), id, hugePNGHeader(), version, time.Now())
		Expect(stored("al-1").BlurHash).To(Equal("KEEP"))
	})

	It("does not mark older served bytes as current when the version advances mid-serve", func() {
		// The cover was replaced and scanned after this serve started: the clamp must stop at the
		// serve's start, so the DTO keeps omitting until the new bytes are served.
		changedAt := version.Add(time.Hour)
		id := album(model.Album{ID: "al-1", UpdatedAt: changedAt})
		u.update(GinkgoT().Context(), id, realPNGBytes("old-bytes"), version, version)
		al := stored("al-1")
		Expect(al.BlurHash).ToNot(BeEmpty())
		Expect(al.BlurHashUpdatedAt).To(HaveValue(Equal(version)))
		Expect(al.BlurHashUpdatedAt.Before(al.ArtworkUpdatedAt())).To(BeTrue(), "must read as stale")
	})

	It("does not write when a placeholder is served and nothing was ever stored", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		u.update(GinkgoT().Context(), id, placeholderImages()[0], version, time.Now())
		Expect(stored("al-1").BlurHashUpdatedAt).To(BeNil())
	})

	It("ignores non-eligible artwork kinds", func() {
		Expect(func() {
			u.clearIfStored(GinkgoT().Context(), model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: "mf-1"})
		}).ToNot(Panic())
		Expect(u.seen).To(BeEmpty())
	})

	It("keys the decode cache by identity, ignoring the artwork id's embedded timestamp", func() {
		id := album(model.Album{ID: "al-1", UpdatedAt: version})
		data := realPNGBytes("dedup")
		u.update(GinkgoT().Context(), id, data, version, time.Now())
		bumped := id
		bumped.LastUpdate = version.Add(time.Hour)
		u.update(GinkgoT().Context(), bumped, data, version, time.Now())
		Expect(u.seen).To(HaveLen(1))
	})
})
