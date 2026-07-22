package originals_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/core/artwork/originals"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Originals Store", func() {
	var store *originals.Store
	var root string

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		store = originals.New(root)
	})

	It("hashes deterministically", func() {
		h1, err := originals.Hash(bytes.NewReader([]byte("some image bytes")))
		Expect(err).ToNot(HaveOccurred())
		h2, _ := originals.Hash(bytes.NewReader([]byte("some image bytes")))
		Expect(h1).To(Equal(h2))
		Expect(h1).To(HaveLen(16))
		h3, _ := originals.Hash(bytes.NewReader([]byte("other bytes")))
		Expect(h3).ToNot(Equal(h1))
	})

	It("writes sharded and reads back", func() {
		data := []byte("jpeg-bytes")
		h, _ := originals.Hash(bytes.NewReader(data))
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
		h, _ := originals.Hash(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
	})

	It("returns fs.ErrNotExist for missing images", func() {
		_, err := store.Open("beefbeefbeefbeef", "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("removes without error when already gone", func() {
		Expect(store.Remove("beefbeefbeefbeef", "image/jpeg")).To(Succeed())
	})

	It("sweeps unknown files, keeps known ones", func() {
		d1 := []byte("keep-me")
		h1, _ := originals.Hash(bytes.NewReader(d1))
		Expect(store.Write(h1, "image/jpeg", bytes.NewReader(d1))).To(Succeed())
		d2 := []byte("orphan")
		h2, _ := originals.Hash(bytes.NewReader(d2))
		Expect(store.Write(h2, "image/jpeg", bytes.NewReader(d2))).To(Succeed())

		removed, err := store.Sweep(func(h string) bool { return h == h1 })
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(Equal(1))
		_, err = store.Open(h2, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(h1, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})
})
