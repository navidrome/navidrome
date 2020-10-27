package subsonic

import (
	"context"
	"net/http/httptest"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AlbumListController", func() {
	var controller *AlbumListController
	var ds model.DataStore
	var mockRepo *persistence.MockAlbum
	var w *httptest.ResponseRecorder
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &persistence.MockDataStore{}
		mockRepo = ds.Album(ctx).(*persistence.MockAlbum)
		controller = NewAlbumListController(ds, nil)
		w = httptest.NewRecorder()
	})

	Describe("GetAlbumList", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			mockRepo.SetData(`[{"id": "1"},{"id": "2"}]`)
			resp, err := controller.GetAlbumList(w, r)

			Expect(err).To(BeNil())
			Expect(resp.AlbumList.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList.Album[1].Id).To(Equal("2"))
			Expect(mockRepo.Options.Offset).To(Equal(10))
			Expect(mockRepo.Options.Max).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()
			_, err := controller.GetAlbumList(w, r)

			Expect(err).To(MatchError("Required string parameter 'type' is not present"))
		})

		It("should return error if call fails", func() {
			mockRepo.SetError(true)
			r := newGetRequest("type=newest")

			_, err := controller.GetAlbumList(w, r)

			Expect(err).ToNot(BeNil())
		})
	})

	Describe("GetAlbumList2", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			mockRepo.SetData(`[{"id": "1"},{"id": "2"}]`)
			resp, err := controller.GetAlbumList2(w, r)

			Expect(err).To(BeNil())
			Expect(resp.AlbumList2.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList2.Album[1].Id).To(Equal("2"))
			Expect(mockRepo.Options.Offset).To(Equal(10))
			Expect(mockRepo.Options.Max).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()
			_, err := controller.GetAlbumList2(w, r)

			Expect(err).To(MatchError("Required string parameter 'type' is not present"))
		})

		It("should return error if call fails", func() {
			mockRepo.SetError(true)
			r := newGetRequest("type=newest")

			_, err := controller.GetAlbumList2(w, r)

			Expect(err).ToNot(BeNil())
		})
	})
})
