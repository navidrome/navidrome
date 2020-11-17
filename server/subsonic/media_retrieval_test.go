package subsonic

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http/httptest"

	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRetrievalController", func() {
	var controller *MediaRetrievalController
	var artwork *fakeArtwork
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		artwork = &fakeArtwork{}
		controller = NewMediaRetrievalController(artwork, &tests.MockDataStore{})
		w = httptest.NewRecorder()
	})

	Describe("GetCoverArt", func() {
		It("should return data for that id", func() {
			artwork.data = "image data"
			r := newGetRequest("id=34", "size=128")
			_, err := controller.GetCoverArt(w, r)

			Expect(err).To(BeNil())
			Expect(artwork.recvId).To(Equal("34"))
			Expect(artwork.recvSize).To(Equal(128))
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should fail if missing id parameter", func() {
			r := newGetRequest()
			_, err := controller.GetCoverArt(w, r)

			Expect(err).To(MatchError("required 'id' parameter is missing"))
		})

		It("should fail when the file is not found", func() {
			artwork.err = model.ErrNotFound
			r := newGetRequest("id=34", "size=128")
			_, err := controller.GetCoverArt(w, r)

			Expect(err).To(MatchError("Artwork not found"))
		})

		It("should fail when there is an unknown error", func() {
			artwork.err = errors.New("weird error")
			r := newGetRequest("id=34", "size=128")
			_, err := controller.GetCoverArt(w, r)

			Expect(err).To(MatchError("weird error"))
		})
	})
})

type fakeArtwork struct {
	data     string
	err      error
	recvId   string
	recvSize int
}

func (c *fakeArtwork) Get(ctx context.Context, id string, size int) (io.ReadCloser, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.recvId = id
	c.recvSize = size
	return ioutil.NopCloser(bytes.NewReader([]byte(c.data))), nil
}
