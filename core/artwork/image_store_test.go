package artwork

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ImageStore", func() {
	var store *ImageStore
	var root string

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		store = NewImageStore(root)
	})

	It("hashes deterministically", func() {
		h1, err := HashImage(bytes.NewReader([]byte("some image bytes")))
		Expect(err).ToNot(HaveOccurred())
		h2, _ := HashImage(bytes.NewReader([]byte("some image bytes")))
		Expect(h1).To(Equal(h2))
		Expect(h1).To(HaveLen(16))
		h3, _ := HashImage(bytes.NewReader([]byte("other bytes")))
		Expect(h3).ToNot(Equal(h1))
	})

	It("writes sharded and reads back", func() {
		data := []byte("jpeg-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())

		Expect(filepath.Join(root, h[0:2], h[2:4], h+".jpg")).To(BeAnExistingFile())

		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		defer rc.Close()
		got, _ := io.ReadAll(rc)
		Expect(got).To(Equal(data))
	})

	It("is idempotent on duplicate writes", func() {
		data := []byte("dup")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
	})

	It("refreshes the mtime on a duplicate write", func() {
		data := []byte("touch-me")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/png"), old, old)).To(Succeed())

		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())

		info, err := os.Stat(store.path(h, "image/png"))
		Expect(err).ToNot(HaveOccurred())
		Expect(info.ModTime()).To(BeTemporally(">", time.Now().Add(-time.Minute)))
	})

	It("rewrites the bytes when the existing file vanished before the liveness touch", func() {
		data := []byte("vanishing")
		h, _ := HashImage(bytes.NewReader(data))
		for range 10 {
			Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
			Expect(os.Remove(store.path(h, "image/png"))).To(Succeed())
			Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())

			rc, err := store.Open(h, "image/png")
			Expect(err).ToNot(HaveOccurred())
			got, _ := io.ReadAll(rc)
			rc.Close()
			Expect(got).To(Equal(data))
		}
	})

	It("returns fs.ErrNotExist for missing images", func() {
		_, err := store.Open("beefbeefbeefbeef", "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("removes without error when already gone", func() {
		Expect(store.Remove("beefbeefbeefbeef", "image/jpeg", time.Now())).To(Succeed())
	})

	It("spares a file newer than the cutoff, removes an aged one", func() {
		fresh := []byte("fresh")
		hf, _ := HashImage(bytes.NewReader(fresh))
		Expect(store.Write(hf, "image/jpeg", bytes.NewReader(fresh))).To(Succeed())

		aged := []byte("aged")
		ha, _ := HashImage(bytes.NewReader(aged))
		Expect(store.Write(ha, "image/jpeg", bytes.NewReader(aged))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(ha, "image/jpeg"), old, old)).To(Succeed())

		cutoff := time.Now().Add(-time.Hour)
		Expect(store.Remove(hf, "image/jpeg", cutoff)).To(Succeed())
		Expect(store.Remove(ha, "image/jpeg", cutoff)).To(Succeed())

		rc, err := store.Open(hf, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
		_, err = store.Open(ha, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("sweeps unknown files, keeps known ones", func() {
		d1 := []byte("keep-me")
		h1, _ := HashImage(bytes.NewReader(d1))
		Expect(store.Write(h1, "image/jpeg", bytes.NewReader(d1))).To(Succeed())
		d2 := []byte("orphan")
		h2, _ := HashImage(bytes.NewReader(d2))
		Expect(store.Write(h2, "image/jpeg", bytes.NewReader(d2))).To(Succeed())

		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h2, "image/jpeg"), old, old)).To(Succeed())

		removed, err := store.Sweep(time.Now().Add(-time.Hour), func(h, _ string) bool { return h == h1 })
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(1))
		_, err = store.Open(h2, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(h1, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("sweeps a stale mime variant of a known hash, keeps the current one", func() {
		data := []byte("same-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/png"), old, old)).To(Succeed())
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())

		// The recorded mime is image/jpeg, so the .png variant is obsolete.
		removed, err := store.Sweep(time.Now().Add(-time.Hour), func(hash, ext string) bool {
			return hash == h && ext == ".jpg"
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(1))
		_, err = store.Open(h, "image/png")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("keeps young unknown files inside the grace window", func() {
		d := []byte("fresh-orphan")
		h, _ := HashImage(bytes.NewReader(d))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(d))).To(Succeed())

		removed, err := store.Sweep(time.Now().Add(-time.Hour), func(string, string) bool { return false })
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(0))
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("removes abandoned temp files past the grace window, keeps fresh ones", func() {
		oldTmp := filepath.Join(root, ".old.tmp")
		Expect(os.WriteFile(oldTmp, []byte("x"), 0600)).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(oldTmp, old, old)).To(Succeed())

		freshTmp := filepath.Join(root, ".fresh.tmp")
		Expect(os.WriteFile(freshTmp, []byte("y"), 0600)).To(Succeed())

		removed, err := store.Sweep(time.Now().Add(-time.Hour), func(string, string) bool { return true })
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(1))
		Expect(oldTmp).ToNot(BeAnExistingFile())
		Expect(freshTmp).To(BeAnExistingFile())
	})
})
