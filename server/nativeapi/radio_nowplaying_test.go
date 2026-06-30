package nativeapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	coreradio "github.com/navidrome/navidrome/core/radio"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Radio now-playing metadata endpoints", func() {
	var (
		router      http.Handler
		ds          *tests.MockDataStore
		radioRepo   *tests.MockedRadioRepo
		userRepo    *tests.MockedUserRepo
		manager     *fakeRadioMetadataManager
		testUser    model.User
		authRequest func(method, path string) *http.Request
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.SessionTimeout = time.Minute

		radioRepo = &tests.MockedRadioRepo{
			Data: map[string]*model.Radio{
				"rd-1": {
					ID:        "rd-1",
					Name:      "Test Radio",
					StreamUrl: "https://stream.example.test/radio",
				},
			},
		}
		userRepo = tests.CreateMockUserRepo()
		ds = &tests.MockDataStore{
			MockedRadio:    radioRepo,
			MockedUser:     userRepo,
			MockedProperty: &tests.MockedPropertyRepo{},
		}
		auth.Init(ds)

		testUser = model.User{
			ID:          "user-1",
			UserName:    "testuser",
			Name:        "Test User",
			IsAdmin:     false,
			NewPassword: "testpass",
		}
		Expect(userRepo.Put(&testUser)).To(Succeed())

		manager = &fakeRadioMetadataManager{}
		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil, nil, manager)
		router = server.JWTVerifier(nativeRouter)

		authRequest = func(method, path string) *http.Request {
			req := httptest.NewRequest(method, path, nil)
			token, err := auth.CreateToken(&testUser)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
			return req.WithContext(request.WithClientUniqueId(req.Context(), "client-1"))
		}
	})

	It("starts metadata reading for a stored radio stream URL", func() {
		w := httptest.NewRecorder()
		req := authRequest(http.MethodPost, "/radio/rd-1/nowplaying")

		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(manager.starts).To(Equal([]metadataStart{{
			sessionID: "user-1:client-1",
			station: coreradio.Station{
				ID:        "rd-1",
				StreamURL: "https://stream.example.test/radio",
			},
		}}))
	})

	It("returns not found when starting metadata for a missing radio", func() {
		w := httptest.NewRecorder()
		req := authRequest(http.MethodPost, "/radio/missing/nowplaying")

		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(manager.starts).To(BeEmpty())
	})

	It("returns unauthorized when request is not authenticated", func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/radio/rd-1/nowplaying", nil)

		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(manager.starts).To(BeEmpty())
	})

	It("stops metadata reading idempotently for the active session", func() {
		w := httptest.NewRecorder()
		req := authRequest(http.MethodDelete, "/radio/rd-1/nowplaying")

		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(manager.stops).To(Equal([]string{"user-1:client-1"}))
	})
})

type metadataStart struct {
	sessionID string
	station   coreradio.Station
}

type fakeRadioMetadataManager struct {
	starts []metadataStart
	stops  []string
}

func (m *fakeRadioMetadataManager) Start(_ context.Context, sessionID string, station coreradio.Station) error {
	m.starts = append(m.starts, metadataStart{sessionID: sessionID, station: station})
	return nil
}

func (m *fakeRadioMetadataManager) Stop(sessionID string) {
	m.stops = append(m.stops, sessionID)
}
