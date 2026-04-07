package subsonic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetOpenSubsonicExtensions", func() {
	var (
		w *httptest.ResponseRecorder
		r *http.Request
	)

	BeforeEach(func() {
		w = httptest.NewRecorder()
		r = httptest.NewRequest("GET", "/getOpenSubsonicExtensions?f=json", nil)
	})

	Context("when ffprobe is available", func() {
		It("includes all extensions including transcoding", func() {
			mock := &mockTranscodeDecision{probeAvailable: true}
			router := New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mock)
			router.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
			var response responses.JsonWrapper
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(*response.Subsonic.OpenSubsonicExtensions).To(SatisfyAll(
				HaveLen(5),
				ContainElement(responses.OpenSubsonicExtension{Name: "transcodeOffset", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "formPost", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "songLyrics", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "indexBasedQueue", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "transcoding", Versions: []int32{1}}),
			))
		})
	})

	Context("when ffprobe is not available", func() {
		It("omits the transcoding extension", func() {
			mock := &mockTranscodeDecision{probeAvailable: false}
			router := New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mock)
			router.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			var response responses.JsonWrapper
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(*response.Subsonic.OpenSubsonicExtensions).To(SatisfyAll(
				HaveLen(4),
				ContainElement(responses.OpenSubsonicExtension{Name: "transcodeOffset", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "formPost", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "songLyrics", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "indexBasedQueue", Versions: []int32{1}}),
			))
		})
	})
})
