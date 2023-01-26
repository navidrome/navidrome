package subsonic

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRetrievalController", func() {
	var router *Router
	var ds model.DataStore
	mockRepo := &mockedMediaFile{}
	var artwork *fakeArtwork
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		ds = &tests.MockDataStore{
			MockedMediaFile: mockRepo,
		}
		artwork = &fakeArtwork{}
		router = New(ds, artwork, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w = httptest.NewRecorder()
	})

	Describe("GetCoverArt", func() {
		It("should return data for that id", func() {
			artwork.data = "image data"
			r := newGetRequest("id=34", "size=128")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(BeNil())
			Expect(artwork.recvId).To(Equal("34"))
			Expect(artwork.recvSize).To(Equal(128))
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should return placeholder if id parameter is missing (mimicking Subsonic)", func() {
			r := newGetRequest()
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(BeNil())
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should fail when the file is not found", func() {
			artwork.err = model.ErrNotFound
			r := newGetRequest("id=34", "size=128")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(MatchError("Artwork not found"))
		})

		It("should fail when there is an unknown error", func() {
			artwork.err = errors.New("weird error")
			r := newGetRequest("id=34", "size=128")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(MatchError("weird error"))
		})
	})

	Describe("GetLyrics", func() {
		It("should return data for given artist & title", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			mockRepo.SetData(model.MediaFiles{
				{
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
					Lyrics: "[00:18.80]We're no strangers to love\n[00:22.80]You know the rules and so do I",
				},
			})
			response, err := router.GetLyrics(r)
			if err != nil {
				log.Error("You're missing something.", err)
			}
			Expect(err).To(BeNil())
			Expect(response.Lyrics.Artist).To(Equal("Rick Astley"))
			Expect(response.Lyrics.Title).To(Equal("Never Gonna Give You Up"))
			Expect(response.Lyrics.Value).To(Equal("We're no strangers to love\nYou know the rules and so do I"))
		})
		It("should return empty subsonic response if the record corresponding to the given artist & title is not found", func() {
			r := newGetRequest("artist=Dheeraj", "title=Rinkiya+Ke+Papa")
			mockRepo.SetData(model.MediaFiles{})
			response, err := router.GetLyrics(r)
			if err != nil {
				log.Error("You're missing something.", err)
			}
			Expect(err).To(BeNil())
			Expect(response.Lyrics.Artist).To(Equal(""))
			Expect(response.Lyrics.Title).To(Equal(""))
			Expect(response.Lyrics.Value).To(Equal(""))

		})
	})
})

type fakeArtwork struct {
	data     string
	err      error
	recvId   string
	recvSize int
}

func (c *fakeArtwork) Get(_ context.Context, id string, size int) (io.ReadCloser, time.Time, error) {
	if c.err != nil {
		return nil, time.Time{}, c.err
	}
	c.recvId = id
	c.recvSize = size
	return io.NopCloser(bytes.NewReader([]byte(c.data))), time.Time{}, nil
}

var _ = Describe("isSynced", func() {
	It("returns false if lyrics contain no timestamps", func() {
		Expect(isSynced("Just in case my car goes off the highway")).To(Equal(false))
		Expect(isSynced("[02.50] Just in case my car goes off the highway")).To(Equal(false))
	})
	It("returns false if lyrics is an empty string", func() {
		Expect(isSynced("")).To(Equal(false))
	})
	It("returns true if lyrics contain timestamps", func() {
		Expect(isSynced(`NF Real Music
		[00:00] First line
		[00:00.85] JUST LIKE YOU
		[00:00.85] Just in case my car goes off the highway`)).To(Equal(true))
		Expect(isSynced("[04:02:50.85] Never gonna give you up")).To(Equal(true))
		Expect(isSynced("[02:50.85] Never gonna give you up")).To(Equal(true))
		Expect(isSynced("[02:50] Never gonna give you up")).To(Equal(true))
	})

})

type mockedMediaFile struct {
	model.MediaFileRepository
	data model.MediaFiles
}

func (m *mockedMediaFile) SetData(mfs model.MediaFiles) {
	m.data = mfs
}

func (m *mockedMediaFile) GetAll(...model.QueryOptions) (model.MediaFiles, error) {
	return m.data, nil
}
