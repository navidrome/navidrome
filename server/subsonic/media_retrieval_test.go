package subsonic

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"slices"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRetrievalController", func() {
	var router *Router
	var ds model.DataStore
	mockRepo := &mockedMediaFile{MockMediaFileRepo: tests.MockMediaFileRepo{}}
	var artwork *fakeArtwork
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		ds = &tests.MockDataStore{
			MockedMediaFile: mockRepo,
		}
		artwork = &fakeArtwork{data: "image data"}
		router = New(ds, artwork, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, lyrics.NewLyrics(ds, nil), nil, nil)
		w = httptest.NewRecorder()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.LyricsPriority = "embedded,.lrc"
	})

	Describe("GetCoverArt", func() {
		It("should return data for that id", func() {
			r := newGetRequest("id=34", "size=128", "square=true")
			_, err := router.GetCoverArt(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(artwork.recvSize).To(Equal(128))
			Expect(artwork.recvSquare).To(BeTrue())
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should return placeholder if id parameter is missing (mimicking Subsonic)", func() {
			r := newGetRequest() // No id parameter
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(BeNil())
			Expect(artwork.recvId).To(BeEmpty())
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should fail when the file is not found", func() {
			artwork.err = model.ErrNotFound
			r := newGetRequest("id=34", "size=128", "square=true")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(MatchError("Artwork not found"))
		})

		It("should fail when there is an unknown error", func() {
			artwork.err = errors.New("weird error")
			r := newGetRequest("id=34", "size=128")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(MatchError("weird error"))
		})

		When("client disconnects (context is cancelled)", func() {
			It("should not call the service if cancelled before the call", func() {
				ctx, cancel := context.WithCancel(context.Background())
				r := newGetRequest("id=34", "size=128", "square=true")
				r = r.WithContext(ctx)
				cancel()

				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(artwork.recvId).To(Equal(""))
				Expect(artwork.recvSize).To(Equal(0))
				Expect(artwork.recvSquare).To(BeFalse())
				Expect(w.Body.String()).To(BeEmpty())
			})

			It("should not return data if cancelled during the call", func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				r := newGetRequest("id=34", "size=128", "square=true")
				r = r.WithContext(ctx)
				artwork.ctxCancelFunc = cancel

				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(artwork.recvId).To(Equal("34"))
				Expect(artwork.recvSize).To(Equal(128))
				Expect(artwork.recvSquare).To(BeTrue())
				Expect(w.Body.String()).To(BeEmpty())
			})
		})
	})

	Describe("GetLyrics", func() {
		It("should return data for given artist & title", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			lyrics, _ := model.ToLyrics("eng", "[00:18.80]We're no strangers to love\n[00:22.80]You know the rules and so do I")
			lyricsJson, err := json.Marshal(model.LyricList{
				*lyrics,
			})
			Expect(err).ToNot(HaveOccurred())

			mockRepo.SetData(model.MediaFiles{
				{
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
					Lyrics: string(lyricsJson),
				},
			})
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Artist).To(Equal("Rick Astley"))
			Expect(response.Lyrics.Title).To(Equal("Never Gonna Give You Up"))
			Expect(response.Lyrics.Value).To(Equal("We're no strangers to love\nYou know the rules and so do I\n"))
		})
		It("should surface the main-kind track when translation tracks are present", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			start := int64(0)
			lyricsJSON, err := json.Marshal(model.LyricList{
				{Kind: model.LyricKindTranslation, Lang: "por", Line: []model.Line{{Start: &start, Value: "Nunca vou te decepcionar"}}},
				{Kind: model.LyricKindMain, Lang: "eng", Line: []model.Line{{Start: &start, Value: "Never gonna let you down"}}},
			})
			Expect(err).ToNot(HaveOccurred())
			mockRepo.SetData(model.MediaFiles{
				{
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
					Lyrics: string(lyricsJSON),
				},
			})
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Value).To(Equal("Never gonna let you down\n"))
		})
		It("should return empty subsonic response if the record corresponding to the given artist & title is not found", func() {
			r := newGetRequest("artist=Dheeraj", "title=Rinkiya+Ke+Papa")
			mockRepo.SetData(model.MediaFiles{})
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Artist).To(Equal(""))
			Expect(response.Lyrics.Title).To(Equal(""))
			Expect(response.Lyrics.Value).To(Equal(""))
		})
		It("should return lyric file when finding mediafile with no embedded lyrics but present on filesystem", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			mockRepo.SetData(model.MediaFiles{
				{
					Path:   "tests/fixtures/test.mp3",
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
				},
			})
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Artist).To(Equal("Rick Astley"))
			Expect(response.Lyrics.Title).To(Equal("Never Gonna Give You Up"))
			Expect(response.Lyrics.Value).To(Equal("We're no strangers to love\nYou know the rules and so do I\n"))
		})
	})
})

type fakeArtwork struct {
	artwork.Artwork
	data          string
	err           error
	ctxCancelFunc func()
	recvId        string
	recvSize      int
	recvSquare    bool
}

func (c *fakeArtwork) GetOrPlaceholder(_ context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error) {
	if c.err != nil {
		return nil, time.Time{}, c.err
	}
	c.recvId = id
	c.recvSize = size
	c.recvSquare = square
	if c.ctxCancelFunc != nil {
		c.ctxCancelFunc()
		return nil, time.Time{}, context.Canceled
	}
	return io.NopCloser(bytes.NewReader([]byte(c.data))), time.Time{}, nil
}

type mockedMediaFile struct {
	tests.MockMediaFileRepo
}

func (m *mockedMediaFile) GetAll(opts ...model.QueryOptions) (model.MediaFiles, error) {
	data, err := m.MockMediaFileRepo.GetAll(opts...)
	if err != nil {
		return nil, err
	}
	if len(opts) == 0 || opts[0].Sort != "lyrics, updated_at" {
		return data, nil
	}

	result := slices.Clone(data)
	slices.SortFunc(result, func(a, b model.MediaFile) int {
		diff := cmp.Or(
			cmp.Compare(a.Lyrics, b.Lyrics),
			cmp.Compare(a.UpdatedAt.Unix(), b.UpdatedAt.Unix()),
		)
		if opts[0].Order == "desc" {
			return -diff
		}
		return diff
	})
	return result, nil
}
