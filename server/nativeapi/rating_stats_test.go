package nativeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rating Stats API", func() {
	var ds *tests.MockDataStore
	var router http.Handler
	var adminUser, regularUser model.User

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil, nil)
		router = server.JWTVerifier(nativeRouter)

		adminUser = model.User{
			ID:          "admin-1",
			UserName:    "admin",
			Name:        "Admin User",
			IsAdmin:     true,
			NewPassword: "adminpass",
		}
		regularUser = model.User{
			ID:          "user-1",
			UserName:    "regular",
			Name:        "Regular User",
			IsAdmin:     false,
			NewPassword: "userpass",
		}

		Expect(ds.User(context.TODO()).Put(&adminUser)).To(Succeed())
		Expect(ds.User(context.TODO()).Put(&regularUser)).To(Succeed())

		mockedUserRepo := ds.MockedUser.(*tests.MockedUserRepo)
		mockedUserRepo.RatingStatsData = []model.UserRatingStats{
			{
				UserID:   "admin-1",
				UserName: "admin",
				SongStats: []model.RatingStat{
					{Rating: 5, Count: 3},
				},
			},
			{
				UserID:   "user-1",
				UserName: "regular",
				SongStats: []model.RatingStat{
					{Rating: 4, Count: 2},
				},
			},
		}
		mockedUserRepo.RatingItemsData = []model.RatedItem{
			{ID: "song-1", Name: "Test Song", Artist: "Test Artist"},
		}
	})

	Describe("GET /ratingStats", func() {
		Context("as admin user", func() {
			It("returns all users stats", func() {
				adminToken, err := auth.CreateToken(&adminUser)
				Expect(err).ToNot(HaveOccurred())

				req := createAuthenticatedRequest("GET", "/ratingStats", nil, adminToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				var stats []model.UserRatingStats
				Expect(json.Unmarshal(w.Body.Bytes(), &stats)).To(Succeed())
				Expect(stats).To(HaveLen(2))
			})
		})

		Context("as regular user", func() {
			It("returns only own stats", func() {
				userToken, err := auth.CreateToken(&regularUser)
				Expect(err).ToNot(HaveOccurred())

				req := createAuthenticatedRequest("GET", "/ratingStats", nil, userToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				var stats []model.UserRatingStats
				Expect(json.Unmarshal(w.Body.Bytes(), &stats)).To(Succeed())
				Expect(stats).To(HaveLen(1))
				Expect(stats[0].UserID).To(Equal("user-1"))
			})
		})

		Context("without authentication", func() {
			It("returns 401", func() {
				req := createUnauthenticatedRequest("GET", "/ratingStats", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

	Describe("GET /ratingItems", func() {
		Context("as admin user", func() {
			It("can query any user's items", func() {
				adminToken, err := auth.CreateToken(&adminUser)
				Expect(err).ToNot(HaveOccurred())

				req := createAuthenticatedRequest("GET", "/ratingItems?userId=user-1&type=song&rating=4", nil, adminToken)
				req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})
		})

		Context("as regular user", func() {
			var userToken string

			BeforeEach(func() {
				var err error
				userToken, err = auth.CreateToken(&regularUser)
				Expect(err).ToNot(HaveOccurred())
			})

			It("can query own items", func() {
				req := createAuthenticatedRequest("GET", "/ratingItems?userId=user-1&type=song&rating=4", nil, userToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})

			It("returns 403 when querying another user's items", func() {
				req := createAuthenticatedRequest("GET", "/ratingItems?userId=admin-1&type=song&rating=5", nil, userToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})

			It("returns 400 for invalid rating", func() {
				req := createAuthenticatedRequest("GET", "/ratingItems?userId=user-1&type=song&rating=9", nil, userToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("returns 400 for invalid type", func() {
				req := createAuthenticatedRequest("GET", "/ratingItems?userId=user-1&type=artist&rating=3", nil, userToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("returns 400 when userId is missing", func() {
				req := createAuthenticatedRequest("GET", "/ratingItems?type=song&rating=3", nil, userToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("without authentication", func() {
			It("returns 401", func() {
				req := createUnauthenticatedRequest("GET", "/ratingItems?userId=user-1&type=song&rating=4", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})
})
