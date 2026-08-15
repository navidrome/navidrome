package nativeapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Media file deletion endpoint", func() {
	var maintenance *mediaDeletionMaintenanceMock

	requestDeletion := func(user model.User) *httptest.ResponseRecorder {
		router := chi.NewRouter()
		router.With(adminOnlyMiddleware).Delete("/song/{id}/file", deleteMediaFile(maintenance))
		req := httptest.NewRequest(http.MethodDelete, "/song/song-1/file", nil)
		req = req.WithContext(request.WithUser(req.Context(), user))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}

	BeforeEach(func() {
		maintenance = &mediaDeletionMaintenanceMock{}
	})

	It("allows an administrator and returns a React Admin compatible response", func() {
		response := requestDeletion(model.User{ID: "admin", IsAdmin: true})

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Body.String()).To(MatchJSON(`{"id":"song-1"}`))
		Expect(maintenance.deletedID).To(Equal("song-1"))
	})

	It("blocks a non-administrator before calling the service", func() {
		response := requestDeletion(model.User{ID: "user", IsAdmin: false})

		Expect(response.Code).To(Equal(http.StatusForbidden))
		Expect(maintenance.deletedID).To(BeEmpty())
	})

	DescribeTable("maps expected failures without exposing internal details",
		func(serviceError error, expectedStatus int) {
			maintenance.err = serviceError
			response := requestDeletion(model.User{ID: "admin", IsAdmin: true})
			Expect(response.Code).To(Equal(expectedStatus))
		},
		Entry("disabled", core.ErrMediaFileDeletionDisabled, http.StatusForbidden),
		Entry("not found", model.ErrNotFound, http.StatusNotFound),
		Entry("read-only storage", core.ErrMediaFileDeletionUnsupported, http.StatusConflict),
		Entry("unexpected error", errors.New("private filesystem detail"), http.StatusInternalServerError),
	)
})

type mediaDeletionMaintenanceMock struct {
	deletedID string
	err       error
}

func (m *mediaDeletionMaintenanceMock) DeleteMediaFile(_ context.Context, id string) error {
	m.deletedID = id
	return m.err
}

func (*mediaDeletionMaintenanceMock) DeleteMissingFiles(context.Context, []string) error {
	return nil
}

func (*mediaDeletionMaintenanceMock) DeleteAllMissingFiles(context.Context) error {
	return nil
}
