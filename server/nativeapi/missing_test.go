package nativeapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Missing Files Endpoint", func() {
	var router http.Handler
	var token string

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.SessionTimeout = time.Minute
		conf.Server.EnableSharing = false

		mfRepo := tests.CreateMockMediaFileRepo()
		mfRepo.SetData(model.MediaFiles{
			{ID: "missing-1", Title: "Gone", Missing: true},
			{ID: "present-1", Title: "Here"},
		})
		userRepo := tests.CreateMockUserRepo()
		ds := &tests.MockDataStore{MockedMediaFile: mfRepo, MockedUser: userRepo, MockedProperty: &tests.MockedPropertyRepo{}}
		auth.Init(ds)

		user := model.User{ID: "user-1", UserName: "user", NewPassword: "pass"}
		Expect(userRepo.Put(&user)).To(Succeed())
		var err error
		token, err = auth.CreateToken(&user)
		Expect(err).ToNot(HaveOccurred())

		router = server.JWTVerifier(New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil, nil, nil))
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
