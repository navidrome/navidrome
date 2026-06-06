package subsonic_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	sonicsvc "github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/server/subsonic"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockSonicPluginLoader struct {
	names []string
}

func (m *mockSonicPluginLoader) PluginNames(capability string) []string {
	if capability == "SonicSimilarity" {
		return m.names
	}
	return nil
}

func (m *mockSonicPluginLoader) LoadSonicSimilarity(_ string) (sonicsvc.Provider, bool) {
	return nil, false
}

var _ = Describe("GetOpenSubsonicExtensions", func() {
	var (
		router *subsonic.Router
		w      *httptest.ResponseRecorder
		r      *http.Request
	)

	JustBeforeEach(func() {
		w = httptest.NewRecorder()
		r = httptest.NewRequest("GET", "/getOpenSubsonicExtensions?f=json", nil)
	})

	Context("without sonic similarity plugin", func() {
		BeforeEach(func() {
			router = subsonic.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		})

		It("should return the base 6 OpenSubsonicExtensions without sonicSimilarity", func() {
			router.ServeHTTP(w, r)

			// Make sure the endpoint is public, by not passing any authentication
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))

			var response responses.JsonWrapper
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(*response.Subsonic.OpenSubsonicExtensions).To(SatisfyAll(
				HaveLen(6),
				ContainElement(responses.OpenSubsonicExtension{Name: "transcodeOffset", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "formPost", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "songLyrics", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "indexBasedQueue", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "transcoding", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "playbackReport", Versions: []int32{1}}),
			))
			Expect(*response.Subsonic.OpenSubsonicExtensions).NotTo(
				ContainElement(responses.OpenSubsonicExtension{Name: "sonicSimilarity", Versions: []int32{1}}),
			)
		})
	})

	Context("with sonic similarity plugin", func() {
		BeforeEach(func() {
			sonicService := sonicsvc.New(nil, &mockSonicPluginLoader{names: []string{"test-plugin"}}, nil)
			router = subsonic.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sonicService)
		})

		It("should return 7 extensions including sonicSimilarity", func() {
			router.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))

			var response responses.JsonWrapper
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(*response.Subsonic.OpenSubsonicExtensions).To(SatisfyAll(
				HaveLen(7),
				ContainElement(responses.OpenSubsonicExtension{Name: "transcodeOffset", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "formPost", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "songLyrics", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "indexBasedQueue", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "transcoding", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "playbackReport", Versions: []int32{1}}),
				ContainElement(responses.OpenSubsonicExtension{Name: "sonicSimilarity", Versions: []int32{1}}),
			))
		})
	})
})
