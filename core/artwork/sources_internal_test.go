package artwork

import (
	"io"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fromExternalFile", func() {
	It("opens a matching file via the library FS", func() {
		fsys := fstest.MapFS{
			"Artist/Album/cover.jpg": &fstest.MapFile{Data: []byte("cover-bytes")},
		}
		f := fromExternalFile(GinkgoT().Context(), fsys, []string{"Artist/Album/cover.jpg"}, "cover.*")
		r, path, err := f()
		Expect(err).ToNot(HaveOccurred())
		defer r.Close()
		b, _ := io.ReadAll(r)
		Expect(b).To(Equal([]byte("cover-bytes")))
		Expect(path).To(Equal("Artist/Album/cover.jpg"))
	})

	It("returns an error when no file matches", func() {
		fsys := fstest.MapFS{
			"Artist/Album/something.txt": &fstest.MapFile{Data: []byte("x")},
		}
		f := fromExternalFile(GinkgoT().Context(), fsys, []string{"Artist/Album/something.txt"}, "cover.*")
		_, _, err := f()
		Expect(err).To(HaveOccurred())
	})

	It("skips files that fail to open and tries the next match", func() {
		fsys := fstest.MapFS{
			"a/cover.jpg": &fstest.MapFile{Data: []byte("a")},
		}
		// "missing/cover.jpg" is in candidates but not in the FS — should be skipped.
		f := fromExternalFile(GinkgoT().Context(), fsys, []string{"missing/cover.jpg", "a/cover.jpg"}, "cover.*")
		r, path, err := f()
		Expect(err).ToNot(HaveOccurred())
		defer r.Close()
		b, _ := io.ReadAll(r)
		Expect(b).To(Equal([]byte("a")))
		Expect(path).To(Equal("a/cover.jpg"))
	})
})
