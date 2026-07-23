package subsonic

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"path/filepath"
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
		albumRepo := &tests.MockAlbumRepo{}
		albumRepo.SetData(model.Albums{{ID: "34"}}) // the id the specs request, made accessible
		ds = &tests.MockDataStore{
			MockedMediaFile: mockRepo,
			MockedAlbum:     albumRepo,
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

		It("serves a placeholder for an entity the caller cannot access", func() {
			// al-99 is not in the (filtered) album repo, so the caller must not get its bytes.
			r := newGetRequest("id=al-99")
			_, err := router.GetCoverArt(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(200))
			Expect(w.Body.String()).ToNot(Equal(artwork.data))
			Expect(w.Header().Get("Cache-Control")).To(Equal("no-store"))
			Expect(w.Header().Get("ETag")).To(BeEmpty())
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
				ctx, cancel := context.WithCancel(GinkgoT().Context())
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
				ctx, cancel := context.WithCancel(GinkgoT().Context())
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

		Describe("caching headers", func() {
			const hash = "0123456789abcdef"

			It("sets an ETag and no-cache for a bare id", func() {
				artwork.hash = hash
				r := newGetRequest("id=al-34")
				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(w.Header().Get("ETag")).To(Equal(`"` + hash + `"`))
				Expect(w.Header().Get("Cache-Control")).To(Equal("public, no-cache"))
				Expect(w.Body.String()).To(Equal(artwork.data))
			})

			It("marks the response immutable when the id asserts the current hash", func() {
				artwork.hash = hash
				r := newGetRequest("id=al-34_" + hash)
				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
			})

			It("returns 304 with no body when If-None-Match matches", func() {
				artwork.hash = hash
				r := newGetRequest("id=al-34")
				r.Header.Set("If-None-Match", `"`+hash+`"`)
				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(w.Code).To(Equal(304))
				Expect(w.Body.Len()).To(BeZero())
			})

			It("never caches a placeholder", func() {
				artwork.placeholder = true
				r := newGetRequest("id=al-missing")
				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(w.Code).To(Equal(200))
				Expect(w.Header().Get("Cache-Control")).To(Equal("no-store"))
				Expect(w.Header().Get("ETag")).To(BeEmpty())
				Expect(w.Body.String()).To(Equal(artwork.data))
			})
		})
	})

	Describe("GetLyrics", func() {
		It("should return data for given artist & title", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			lyricsList, _ := model.ParseLyrics(GinkgoT().Context(), ".lrc", "eng", []byte("[00:18.80]We're no strangers to love\n[00:22.80]You know the rules and so do I"))
			lyrics, _ := lyricsList.Main()
			lyricsJson, err := json.Marshal(model.LyricList{
				lyrics,
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
			fixturesDir, err := filepath.Abs("tests/fixtures")
			Expect(err).ToNot(HaveOccurred())
			mockRepo.SetData(model.MediaFiles{
				{
					LibraryPath: fixturesDir,
					Path:        "test.mp3",
					ID:          "1",
					Artist:      "Rick Astley",
					Title:       "Never Gonna Give You Up",
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
	artwork.Service
	data          string
	hash          string
	lastUpdated   time.Time
	placeholder   bool
	err           error
	ctxCancelFunc func()
	recvId        string
	recvSize      int
	recvSquare    bool
}

func (c *fakeArtwork) GetOrPlaceholder(_ context.Context, id string, size int, square bool) (*artwork.Image, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.recvId = id
	c.recvSize = size
	c.recvSquare = square
	if c.ctxCancelFunc != nil {
		c.ctxCancelFunc()
		return nil, context.Canceled
	}
	return &artwork.Image{
		ReadCloser:  io.NopCloser(bytes.NewReader([]byte(c.data))),
		Hash:        c.hash,
		LastUpdated: c.lastUpdated,
		Placeholder: c.placeholder,
	}, nil
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
