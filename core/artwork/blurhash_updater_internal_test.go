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
	var version time.Time

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		u = &blurHashUpdater{
			a:       &artwork{ds: ds},
			buffer:  make(map[model.ArtworkID]blurHashJob),
			wake:    make(chan struct{}, 1),
			last:    make(map[model.ArtworkID]string),
			started: true,
		}
		version = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	It("persists a hash computed from the given bytes", func() {
		al := model.Album{ID: "al-1", UpdatedAt: version}
		repo := tests.CreateMockAlbumRepo()
		repo.SetData(model.Albums{al})
		ds.MockedAlbum = repo

		u.processJob(GinkgoT().Context(), al.CoverArtID(), blurHashJob{data: realPNGBytes("x"), version: version})
		stored, err := ds.Album(GinkgoT().Context()).Get("al-1")
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.BlurHash).ToNot(BeEmpty())
	})

	It("clears the hash when the bytes are a placeholder", func() {
		al := model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "OLD"}
		repo := tests.CreateMockAlbumRepo()
		repo.SetData(model.Albums{al})
		ds.MockedAlbum = repo

		u.processJob(GinkgoT().Context(), al.CoverArtID(), blurHashJob{data: placeholderImages()[0], version: version})
		stored, _ := ds.Album(GinkgoT().Context()).Get("al-1")
		Expect(stored.BlurHash).To(BeEmpty()) // cleared: placeholder means gone
	})

	It("leaves the hash untouched on undecodable bytes", func() {
		al := model.Album{ID: "al-1", UpdatedAt: version, BlurHash: "KEEP"}
		repo := tests.CreateMockAlbumRepo()
		repo.SetData(model.Albums{al})
		ds.MockedAlbum = repo

		u.processJob(GinkgoT().Context(), al.CoverArtID(), blurHashJob{data: []byte("not an image"), version: version})
		stored, _ := ds.Album(GinkgoT().Context()).Get("al-1")
		Expect(stored.BlurHash).To(Equal("KEEP"))
	})

	It("skips a redundant write when the hash is unchanged (in-memory dedup)", func() {
		al := model.Album{ID: "al-1", UpdatedAt: version}
		repo := tests.CreateMockAlbumRepo()
		repo.SetData(model.Albums{al})
		ds.MockedAlbum = repo
		data := realPNGBytes("dedup")

		u.processJob(GinkgoT().Context(), al.CoverArtID(), blurHashJob{data: data, version: version})
		first, _ := ds.Album(GinkgoT().Context()).Get("al-1")
		u.processJob(GinkgoT().Context(), al.CoverArtID(), blurHashJob{data: data, version: version.Add(time.Hour)})
		second, _ := ds.Album(GinkgoT().Context()).Get("al-1")
		Expect(second.BlurHash).To(Equal(first.BlurHash))
	})

	It("ignores non-eligible artwork kinds on enqueue", func() {
		u.EnqueueBytes(model.ArtworkID{Kind: model.KindMediaFileArtwork, ID: "mf-1"}, realPNGBytes("x"), version)
		Expect(u.buffer).To(BeEmpty())
	})

	It("supersedes a pending gone-check with a bytes job", func() {
		id := model.Album{ID: "al-1"}.CoverArtID()
		u.EnqueueClearIfGone(id, version)
		u.EnqueueBytes(id, realPNGBytes("x"), version.Add(time.Hour))
		Expect(u.buffer[id].checkGone).To(BeFalse())
		Expect(u.buffer[id].data).ToNot(BeNil())
	})
})
