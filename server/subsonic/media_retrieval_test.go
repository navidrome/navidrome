package subsonic

import (
	"errors"
	"io"
	"net/http/httptest"

	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeCover struct {
	data     string
	err      error
	recvId   string
	recvSize int
}

func (c *fakeCover) Get(id string, size int, out io.Writer) error {
	if c.err != nil {
		return c.err
	}
	c.recvId = id
	c.recvSize = size
	out.Write([]byte(c.data))
	return nil
}

var _ = Describe("MediaRetrievalController", func() {
	var controller *MediaRetrievalController
	var cover *fakeCover
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		cover = &fakeCover{}
		controller = NewMediaRetrievalController(cover)
		w = httptest.NewRecorder()
	})

	Describe("GetCoverArt", func() {
		It("should return data for that id", func() {
			cover.data = "image data"
			r := newTestRequest("id=34", "size=128")
			_, err := controller.GetCoverArt(w, r)

			Expect(err).To(BeNil())
			Expect(cover.recvId).To(Equal("34"))
			Expect(cover.recvSize).To(Equal(128))
			Expect(w.Body.String()).To(Equal(cover.data))
		})

		It("should fail if missing id parameter", func() {
			r := newTestRequest()
			_, err := controller.GetCoverArt(w, r)

			Expect(err).To(MatchError("id parameter required"))
		})

		It("should fail when the file is not found", func() {
			cover.err = model.ErrNotFound
			r := newTestRequest("id=34", "size=128")
			_, err := controller.GetCoverArt(w, r)

			Expect(err).To(MatchError("Cover not found"))
		})

		It("should fail when there is an unknown error", func() {
			cover.err = errors.New("weird error")
			r := newTestRequest("id=34", "size=128")
			_, err := controller.GetCoverArt(w, r)

			Expect(err).To(MatchError("Internal Error"))
		})
	})
})
