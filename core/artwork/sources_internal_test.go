package artwork

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
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

var _ = Describe("fromTag", func() {
	It("opens an embedded image via fs.FS", func() {
		fsys := os.DirFS("tests/fixtures/artist/an-album")
		f := fromTag(GinkgoT().Context(), fsys, "test.mp3")
		r, path, err := f()
		Expect(err).ToNot(HaveOccurred())
		defer r.Close()
		Expect(path).To(Equal("test.mp3"))
		b, _ := io.ReadAll(r)
		Expect(b).ToNot(BeEmpty())
	})

	It("returns nil reader when the relative path is empty", func() {
		f := fromTag(GinkgoT().Context(), os.DirFS("."), "")
		r, _, err := f()
		Expect(err).ToNot(HaveOccurred())
		Expect(r).To(BeNil())
	})

	It("errors when the FS file is not seekable", func() {
		fsys := nonSeekableFS{data: []byte("garbage")}
		f := fromTag(GinkgoT().Context(), fsys, "x.mp3")
		_, _, err := f()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not seekable"))
	})
})

// nonSeekableFS is a single-file fs.FS whose Open returns a non-seekable file.
type nonSeekableFS struct{ data []byte }

func (n nonSeekableFS) Open(name string) (fs.File, error) {
	return &nonSeekableFile{r: bytes.NewReader(n.data)}, nil
}

type nonSeekableFile struct{ r *bytes.Reader }

func (n *nonSeekableFile) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n *nonSeekableFile) Close() error               { return nil }
func (n *nonSeekableFile) Stat() (fs.FileInfo, error) { return nil, errors.New("not implemented") }
