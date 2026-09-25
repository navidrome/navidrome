package public

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("handleM3U", func() {
	var ds *tests.MockDataStore
	var shareRepo *tests.MockShareRepo
	var pub *Router

	BeforeEach(func() {
		auth.PublicTokenAuth = jwtauth.New("HS256", []byte("test-secret"), nil)
		ds = &tests.MockDataStore{}
		shareRepo = &tests.MockShareRepo{}
		ds.MockedShare = shareRepo
		pub = &Router{ds: ds, share: core.NewShare(ds)}
	})

	makeRequest := func(id string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/public/"+id+"/m3u?%3Aid="+id, nil)
		w := httptest.NewRecorder()
		pub.handleM3U(w, r)
		return w
	}

	It("sets the M3U content type", func() {
		share := &model.Share{ID: "abc123", Tracks: model.MediaFiles{{ID: "t1", Title: "Track 1"}}}
		shareRepo.ID = share.ID
		shareRepo.Entity = share

		w := makeRequest("abc123")

		Expect(w.Code).To(Equal(http.StatusOK))
		// Result() has the headers sent at WriteHeader time, unlike w.Header()
		Expect(w.Result().Header.Get("Content-Type")).To(Equal("audio/x-mpegurl"))
		Expect(w.Body.String()).To(HavePrefix("#EXTM3U"))
	})

	It("returns 404 when the share does not exist", func() {
		shareRepo.ID = "other"
		shareRepo.Entity = &model.Share{ID: "other"}

		Expect(makeRequest("missing").Code).To(Equal(http.StatusNotFound))
	})
})
