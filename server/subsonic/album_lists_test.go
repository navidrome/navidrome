package subsonic

import (
	"context"
	"errors"
	"net/http/httptest"

	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album Lists", func() {
	var router *Router
	var ds model.DataStore
	var mockRepo *tests.MockAlbumRepo
	var w *httptest.ResponseRecorder
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		mockRepo = ds.Album(ctx).(*tests.MockAlbumRepo)
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w = httptest.NewRecorder()
	})

	Describe("GetAlbumList", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			mockRepo.SetData(model.Albums{
				{ID: "1"}, {ID: "2"},
			})
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList.Album[1].Id).To(Equal("2"))
			Expect(w.Header().Get("x-total-count")).To(Equal("2"))
			Expect(mockRepo.Options.Offset).To(Equal(10))
			Expect(mockRepo.Options.Max).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()
			_, err := router.GetAlbumList(w, r)

			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("should return error if call fails", func() {
			mockRepo.SetError(true)
			r := newGetRequest("type=newest")

			_, err := router.GetAlbumList(w, r)

			Expect(err).To(MatchError(errSubsonic))
			var subErr subError
			errors.As(err, &subErr)
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})
	})

	Describe("GetAlbumList2", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			mockRepo.SetData(model.Albums{
				{ID: "1"}, {ID: "2"},
			})
			resp, err := router.GetAlbumList2(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList2.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList2.Album[1].Id).To(Equal("2"))
			Expect(w.Header().Get("x-total-count")).To(Equal("2"))
			Expect(mockRepo.Options.Offset).To(Equal(10))
			Expect(mockRepo.Options.Max).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()

			_, err := router.GetAlbumList2(w, r)

			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("should return error if call fails", func() {
			mockRepo.SetError(true)
			r := newGetRequest("type=newest")

			_, err := router.GetAlbumList2(w, r)

			Expect(err).To(MatchError(errSubsonic))
			var subErr subError
			errors.As(err, &subErr)
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})
	})
})
