package nativeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Player API key endpoints", func() {
	var repo *tests.MockPlayerRepo
	var router chi.Router

	BeforeEach(func() {
		repo = tests.CreateMockPlayerRepo()
		Expect(repo.Put(&model.Player{ID: "p1", UserId: "u1"})).To(Succeed())
		ds := &tests.MockDataStore{MockedPlayer: repo}
		router = chi.NewRouter()
		router.Post("/player/{id}/apiKey", generatePlayerAPIKey(ds))
		router.Delete("/player/{id}/apiKey", revokePlayerAPIKey(ds))
	})

	serve := func(method, path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w
	}

	It("returns the new key", func() {
		w := serve("POST", "/player/p1/apiKey")
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))

		var body map[string]string
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body["apiKey"]).To(HavePrefix(consts.APIKeyPrefix))
		_, err := repo.FindByAPIKey(body["apiKey"])
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns 404 for an unknown player", func() {
		Expect(serve("POST", "/player/missing/apiKey").Code).To(Equal(http.StatusNotFound))
		Expect(serve("DELETE", "/player/missing/apiKey").Code).To(Equal(http.StatusNotFound))
	})

	It("returns 403 when the user may not change the key", func() {
		repo.Error = rest.ErrPermissionDenied
		Expect(serve("POST", "/player/p1/apiKey").Code).To(Equal(http.StatusForbidden))
		Expect(serve("DELETE", "/player/p1/apiKey").Code).To(Equal(http.StatusForbidden))
	})

	It("revokes the key", func() {
		key, err := repo.GenerateAPIKey("p1")
		Expect(err).ToNot(HaveOccurred())

		Expect(serve("DELETE", "/player/p1/apiKey").Code).To(Equal(http.StatusNoContent))
		_, err = repo.FindByAPIKey(key)
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})
