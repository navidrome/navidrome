package public

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("handleM3U", func() {
	var ctx context.Context
	var ds model.DataStore
	var pub *Router

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		auth.PublicTokenAuth = jwtauth.New("HS256", []byte("test-secret"), nil)
		ds = persistence.New(db.Db())
		pub = &Router{ds: ds, share: core.NewShare(ds)}
	})

	makeRequest := func(id string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/public/"+id+"/m3u?%3Aid="+id, nil)
		w := httptest.NewRecorder()
		pub.handleM3U(w, r)
		return w
	}

	It("sets the M3U content type", func() {
		putMediaFiles(ctx, ds, model.MediaFile{ID: "t1", Title: "Track 1"})
		putShare(ctx, ds, model.Share{ID: "abc123", ResourceType: "media_file", ResourceIDs: "t1"})

		w := makeRequest("abc123")

		Expect(w.Code).To(Equal(http.StatusOK))
		// Result() has the headers sent at WriteHeader time, unlike w.Header()
		Expect(w.Result().Header.Get("Content-Type")).To(Equal("audio/x-mpegurl"))
		Expect(w.Body.String()).To(HavePrefix("#EXTM3U"))
	})

	It("returns 404 when the share does not exist", func() {
		putShare(ctx, ds, model.Share{ID: "other"})

		Expect(makeRequest("missing").Code).To(Equal(http.StatusNotFound))
	})
})
