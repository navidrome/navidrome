package nativeapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Missing Files Endpoint", func() {
	var router http.Handler
	var token string

	BeforeEach(func() {
		conf.Update(func(c *conf.Options) {
			c.SessionTimeout = time.Minute
			c.EnableSharing = false
		})

		ctx := GinkgoT().Context()
		ds := persistence.New(db.Db())
		auth.Init(ctx, ds)

		user := model.User{ID: "user-1", UserName: "user", NewPassword: "pass"}
		putUsers(ctx, ds, &user)
		Expect(ds.MediaFile().Put(ctx, &model.MediaFile{ID: "missing-1", LibraryID: 1, Path: "gone.mp3", Title: "Gone", Missing: true})).To(Succeed())
		Expect(ds.MediaFile().Put(ctx, &model.MediaFile{ID: "present-1", LibraryID: 1, Path: "here.mp3", Title: "Here"})).To(Succeed())
		var err error
		token, err = auth.CreateToken(&user)
		Expect(err).ToNot(HaveOccurred())

		router = server.JWTVerifier(newTestRouter(ds, nil, nil, nil))
	})

	DescribeTable("GET /missing/{id}",
		func(id string, status int) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, createAuthenticatedRequest(http.MethodGet, "/missing/"+id, &bytes.Buffer{}, token))
			Expect(w.Code).To(Equal(status), w.Body.String())
		},
		Entry("returns a missing file", "missing-1", http.StatusOK),
		Entry("returns 404 for a file that is not missing", "present-1", http.StatusNotFound),
		Entry("returns 404 for an unknown id", "unknown", http.StatusNotFound),
	)
})
