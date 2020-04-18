package subsonic

import (
	"context"
	"errors"
	"net/http/httptest"

	"github.com/deluan/navidrome/engine"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeListGen struct {
	engine.ListGenerator
	data       engine.Entries
	err        error
	recvOffset int
	recvSize   int
}

func (lg *fakeListGen) GetAlbums(ctx context.Context, offset int, size int, filter engine.ListFilter) (engine.Entries, error) {
	if lg.err != nil {
		return nil, lg.err
	}
	lg.recvOffset = offset
	lg.recvSize = size
	return lg.data, nil
}

var _ = Describe("AlbumListController", func() {
	var controller *AlbumListController
	var listGen *fakeListGen
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		listGen = &fakeListGen{}
		controller = NewAlbumListController(listGen)
		w = httptest.NewRecorder()
	})

	Describe("GetAlbumList", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			listGen.data = engine.Entries{
				{Id: "1"}, {Id: "2"},
			}
			resp, err := controller.GetAlbumList(w, r)

			Expect(err).To(BeNil())
			Expect(resp.AlbumList.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList.Album[1].Id).To(Equal("2"))
			Expect(listGen.recvOffset).To(Equal(10))
			Expect(listGen.recvSize).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()
			_, err := controller.GetAlbumList(w, r)

			Expect(err).To(MatchError("Required string parameter 'type' is not present"))
		})

		It("should return error if call fails", func() {
			listGen.err = errors.New("some issue")
			r := newGetRequest("type=newest")

			_, err := controller.GetAlbumList(w, r)

			Expect(err).To(MatchError("Internal Error"))
		})
	})

	Describe("GetAlbumList2", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			listGen.data = engine.Entries{
				{Id: "1"}, {Id: "2"},
			}
			resp, err := controller.GetAlbumList2(w, r)

			Expect(err).To(BeNil())
			Expect(resp.AlbumList2.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList2.Album[1].Id).To(Equal("2"))
			Expect(listGen.recvOffset).To(Equal(10))
			Expect(listGen.recvSize).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()
			_, err := controller.GetAlbumList2(w, r)

			Expect(err).To(MatchError("Required string parameter 'type' is not present"))
		})

		It("should return error if call fails", func() {
			listGen.err = errors.New("some issue")
			r := newGetRequest("type=newest")

			_, err := controller.GetAlbumList2(w, r)

			Expect(err).To(MatchError("Internal Error"))
		})
	})
})
