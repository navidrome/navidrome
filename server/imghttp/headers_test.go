package imghttp_test

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/server/imghttp"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const testHash = "0123456789abcdef"
const testRepTag = testHash + ".300.false.q75"

var lastMod = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

func found() *artwork.Image {
	return &artwork.Image{
		ReadCloser:  io.NopCloser(strings.NewReader("IMG")),
		Hash:        testHash,
		LastUpdated: lastMod,
	}
}

func placeholder() *artwork.Image {
	return &artwork.Image{ReadCloser: io.NopCloser(strings.NewReader("PH")), Placeholder: true}
}

// A re-encoded response carries a representation ETag distinct from the pixel hash, so the
// validator versions with the encode settings.
func resized() *artwork.Image {
	return &artwork.Image{
		ReadCloser:  io.NopCloser(strings.NewReader("IMG")),
		Hash:        testHash,
		ETag:        testRepTag,
		LastUpdated: lastMod,
	}
}

func unvalidated() *artwork.Image {
	return &artwork.Image{ReadCloser: io.NopCloser(strings.NewReader("IMG")), LastUpdated: lastMod}
}

var _ = Describe("WriteImageHeaders", func() {
	type testCase struct {
		img           *artwork.Image
		requestedHash string
		ifNoneMatch   string
		want304       bool
		wantCache     string
		wantETag      string
		wantLastMod   bool
	}

	DescribeTable("applies the artwork caching contract",
		func(c testCase) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/img", nil)
			if c.ifNoneMatch != "" {
				r.Header.Set("If-None-Match", c.ifNoneMatch)
			}

			Expect(imghttp.WriteImageHeaders(w, r, c.img, c.requestedHash)).To(Equal(c.want304))

			h := w.Header()
			Expect(h.Get("Cache-Control")).To(Equal(c.wantCache))
			Expect(h.Get("ETag")).To(Equal(c.wantETag))
			if c.img.Placeholder {
				Expect(h.Get("ETag")).To(BeEmpty(), "placeholder must not set an ETag")
				Expect(h.Get("Last-Modified")).To(BeEmpty(), "placeholder must not set Last-Modified")
			}
			if c.wantLastMod {
				Expect(h.Get("Last-Modified")).ToNot(BeEmpty())
			} else {
				Expect(h.Get("Last-Modified")).To(BeEmpty())
			}
			if c.want304 {
				Expect(w.Code).To(Equal(http.StatusNotModified))
				Expect(w.Body.Len()).To(BeZero(), "a 304 must have an empty body")
			}
		},
		Entry("placeholder is never cached and carries no validators",
			testCase{img: placeholder(), wantCache: "no-store"}),
		Entry("found with matching requested hash is immutable",
			testCase{img: found(), requestedHash: testHash, wantCache: "public, max-age=31536000, immutable", wantETag: `"` + testHash + `"`, wantLastMod: true}),
		Entry("found with bare id revalidates via no-cache",
			testCase{img: found(), wantCache: "public, no-cache", wantETag: `"` + testHash + `"`, wantLastMod: true}),
		Entry("found with mismatched requested hash revalidates",
			testCase{img: found(), requestedHash: "ffffffffffffffff", wantCache: "public, no-cache", wantETag: `"` + testHash + `"`, wantLastMod: true}),
		Entry("resized keeps pixel-hash immutable but serves the representation ETag",
			testCase{img: resized(), requestedHash: testHash, wantCache: "public, max-age=31536000, immutable", wantETag: `"` + testRepTag + `"`, wantLastMod: true}),
		Entry("resized 304s on the representation ETag, not the pixel hash",
			testCase{img: resized(), ifNoneMatch: `"` + testRepTag + `"`, want304: true, wantCache: "public, no-cache", wantETag: `"` + testRepTag + `"`, wantLastMod: true}),
		Entry("resized does not 304 on a stale pixel-hash validator (config changed)",
			testCase{img: resized(), ifNoneMatch: `"` + testHash + `"`, want304: false, wantCache: "public, no-cache", wantETag: `"` + testRepTag + `"`, wantLastMod: true}),
		Entry("If-None-Match matching the hash yields 304",
			testCase{img: found(), requestedHash: testHash, ifNoneMatch: `"` + testHash + `"`, want304: true, wantCache: "public, max-age=31536000, immutable", wantETag: `"` + testHash + `"`, wantLastMod: true}),
		Entry("weak If-None-Match matches (weak comparison)",
			testCase{img: found(), ifNoneMatch: `W/"` + testHash + `"`, want304: true, wantCache: "public, no-cache", wantETag: `"` + testHash + `"`, wantLastMod: true}),
		Entry("If-None-Match with multiple values matches one",
			testCase{img: found(), ifNoneMatch: `"deadbeefdeadbeef", W/"` + testHash + `", "cafecafecafecafe"`, want304: true, wantCache: "public, no-cache", wantETag: `"` + testHash + `"`, wantLastMod: true}),
		Entry("If-None-Match star matches any current representation",
			testCase{img: found(), ifNoneMatch: "*", want304: true, wantCache: "public, no-cache", wantETag: `"` + testHash + `"`, wantLastMod: true}),
		Entry("non-matching If-None-Match serves body",
			testCase{img: found(), ifNoneMatch: `"deadbeefdeadbeef"`, want304: false, wantCache: "public, no-cache", wantETag: `"` + testHash + `"`, wantLastMod: true}),
		Entry("placeholder ignores If-None-Match and never 304s",
			testCase{img: placeholder(), ifNoneMatch: "*", want304: false, wantCache: "no-store"}),

		// With no validator, an emitted ETag would be the same empty tag on every such response,
		// and matching it would 304 changed bytes.
		Entry("omits the ETag entirely when there is no validator",
			testCase{img: unvalidated(), wantCache: "public, no-cache", wantLastMod: true}),
		Entry("never 304s an empty validator echoed back by the client",
			testCase{img: unvalidated(), ifNoneMatch: `""`, want304: false,
				wantCache: "public, no-cache", wantLastMod: true}),
	)
})

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type failingFile struct {
	failingReader
	info fs.FileInfo
}

func (f failingFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (failingFile) Close() error { return nil }

var _ = Describe("WriteImage", func() {
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		w = httptest.NewRecorder()
	})

	write := func(r io.Reader) {
		n, err := imghttp.WriteImage(w, &artwork.Image{ReadCloser: io.NopCloser(r)})
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(int64(w.Body.Len())))
	}

	expectWithdrawn := func() {
		h := w.Header()
		Expect(h.Get("Content-Length")).To(BeEmpty())
		Expect(h.Get("ETag")).To(BeEmpty())
		Expect(h.Get("Last-Modified")).To(BeEmpty())
		Expect(h.Get("Cache-Control")).To(Equal("no-store"))
		Expect(w.Body.Len()).To(BeZero())
	}

	It("streams a file of any size with its size as Content-Length", func() {
		data := bytes.Repeat([]byte{0xCD}, 5<<20)
		path := filepath.Join(GinkgoT().TempDir(), "disc1.png")
		Expect(os.WriteFile(path, data, 0600)).To(Succeed())
		f, err := os.Open(path)
		Expect(err).ToNot(HaveOccurred())

		n, err := imghttp.WriteImage(w, &artwork.Image{ReadCloser: f})

		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(int64(len(data))))
		Expect(w.Header().Get("Content-Length")).To(Equal(strconv.Itoa(len(data))))
		Expect(w.Body.Bytes()).To(Equal(data))
	})

	It("streams an embedded fs file with its size as Content-Length", func() {
		f, err := fstest.MapFS{"placeholder.webp": {Data: []byte("PH")}}.Open("placeholder.webp")
		Expect(err).ToNot(HaveOccurred())

		_, err = imghttp.WriteImage(w, &artwork.Image{ReadCloser: f})

		Expect(err).ToNot(HaveOccurred())
		Expect(w.Header().Get("Content-Length")).To(Equal("2"))
		Expect(w.Body.String()).To(Equal("PH"))
	})

	It("streams a cache hit with the section size as Content-Length", func() {
		sr := io.NewSectionReader(strings.NewReader("CACHED"), 0, 6)
		stream := &cache.CachedStream{Reader: sr, Seeker: sr, Cached: true}

		_, err := imghttp.WriteImage(w, &artwork.Image{ReadCloser: stream})

		Expect(err).ToNot(HaveOccurred())
		Expect(w.Header().Get("Content-Length")).To(Equal("6"))
		Expect(w.Body.String()).To(Equal("CACHED"))
	})

	It("buffers a small image of unknown size to send its Content-Length", func() {
		write(strings.NewReader("IMG"))

		Expect(w.Header().Get("Content-Length")).To(Equal("3"))
		Expect(w.Body.String()).To(Equal("IMG"))
	})

	It("streams a large image of unknown size without Content-Length", func() {
		data := bytes.Repeat([]byte{0xAB}, 5<<20)

		write(bytes.NewReader(data))

		Expect(w.Header().Get("Content-Length")).To(BeEmpty())
		Expect(w.Body.Bytes()).To(Equal(data))
	})

	It("writes nothing and withdraws the caching headers when an unknown-size image cannot be read", func() {
		img := found()
		img.ReadCloser = io.NopCloser(failingReader{})
		Expect(imghttp.WriteImageHeaders(w, httptest.NewRequest(http.MethodGet, "/img", nil), img, testHash)).To(BeFalse())

		n, err := imghttp.WriteImage(w, img)

		Expect(err).To(MatchError(io.ErrUnexpectedEOF))
		Expect(n).To(BeZero())
		expectWithdrawn()
	})

	It("writes nothing and withdraws the caching headers when a known-size image cannot be read", func() {
		info, err := fs.Stat(fstest.MapFS{"cover.jpg": {Data: []byte("0123456789")}}, "cover.jpg")
		Expect(err).ToNot(HaveOccurred())
		img := found()
		img.ReadCloser = failingFile{info: info}
		Expect(imghttp.WriteImageHeaders(w, httptest.NewRequest(http.MethodGet, "/img", nil), img, testHash)).To(BeFalse())

		n, err := imghttp.WriteImage(w, img)

		Expect(err).To(MatchError(io.ErrUnexpectedEOF))
		Expect(n).To(BeZero())
		expectWithdrawn()
	})
})
