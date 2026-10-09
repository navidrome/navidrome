package artwork

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/httpclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fromURL", func() {
	var (
		hits   atomic.Int32
		target *httptest.Server
	)

	BeforeEach(func() {
		hits.Store(0)
		target = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
			_, _ = w.Write([]byte("image-bytes"))
		}))
		DeferCleanup(target.Close)
	})

	useClient := func(c *http.Client) {
		prev := remoteImageClient
		remoteImageClient = c
		DeferCleanup(func() { remoteImageClient = prev })
	}
	fetch := func(rawURL string) ([]byte, error) {
		u, err := url.Parse(rawURL)
		Expect(err).ToNot(HaveOccurred())
		r, _, err := fromURL(GinkgoT().Context(), u)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	}
	// Stand-in for a public host: only 127.0.0.1 is allowed, so every other private address stays refused.
	onlyLocalhostV4 := func() *http.Client {
		return httpclient.NewExternal(5*time.Second, netip.MustParsePrefix("127.0.0.1/32"))
	}

	DescribeTable("refuses private and loopback targets as a definitive miss",
		func(rawURL string) {
			useClient(productionImageClient)
			u, _ := url.Parse(target.URL)
			_, err := fetch(strings.ReplaceAll(rawURL, "PORT", u.Port()))
			Expect(err).To(MatchError(model.ErrNotFound))
			Expect(hits.Load()).To(BeZero())
		},
		Entry("IPv4 loopback", "http://127.0.0.1:PORT/x"),
		Entry("localhost", "http://localhost:PORT/x"),
		Entry("cloud metadata", "http://169.254.169.254/"),
		Entry("IPv6 loopback", "http://[::1]/"),
	)

	It("refuses a redirect from an allowed host to a loopback address", func() {
		useClient(onlyLocalhostV4())
		redirector := httptest.NewServer(http.RedirectHandler(strings.Replace(target.URL, "127.0.0.1", "127.0.0.2", 1), http.StatusFound))
		DeferCleanup(redirector.Close)

		_, err := fetch(redirector.URL)
		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(hits.Load()).To(BeZero())
	})

	It("fetches from an allowed address", func() {
		useClient(onlyLocalhostV4())
		Expect(fetch(target.URL + "/cover.jpg")).To(Equal([]byte("image-bytes")))
	})
})

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

	It("skips a matching file that is not an image", func() {
		fsys := fstest.MapFS{
			"a/cover.ini": &fstest.MapFile{Data: []byte("password=secret")},
			"a/cover.jpg": &fstest.MapFile{Data: []byte("a")},
		}
		f := fromExternalFile(GinkgoT().Context(), fsys, []string{"a/cover.ini", "a/cover.jpg"}, "cover.*")
		r, path, err := f()
		Expect(err).ToNot(HaveOccurred())
		defer r.Close()
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
