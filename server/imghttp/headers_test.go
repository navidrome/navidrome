package imghttp_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/server/imghttp"
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

// resized carries a representation ETag distinct from the pixel hash (as a resized/re-encoded
// response does), so the validator versions with the encode settings.
func resized() *artwork.Image {
	return &artwork.Image{
		ReadCloser:  io.NopCloser(strings.NewReader("IMG")),
		Hash:        testHash,
		ETag:        testRepTag,
		LastUpdated: lastMod,
	}
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
			r := httptest.NewRequest("GET", "/img", nil)
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
			Expect(h.Get("Last-Modified") != "").To(Equal(c.wantLastMod))
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
	)
})
