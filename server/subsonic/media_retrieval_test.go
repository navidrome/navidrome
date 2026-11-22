package subsonic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRetrievalController", func() {
	var router *Router
	var ds model.DataStore
	var ctx context.Context
	var artwork *fakeArtwork
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		useRepoRootAsLibrary(ctx, ds)
		artwork = &fakeArtwork{data: "image data"}
		router = New(ds, artwork, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, lyrics.NewLyrics(ds, nil), nil, nil)
		w = httptest.NewRecorder()
		conf.Update(func(c *conf.Options) { c.LyricsPriority = "embedded,.lrc" })
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

			Expect(err).ToNot(HaveOccurred())
			Expect(artwork.recvId).To(BeEmpty())
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		// The service applies the library filter from the caller's context, so elevating here
		// would bypass it.
		It("passes the caller's context to the service rather than elevating", func() {
			r := newGetRequest("id=al-34")
			usr := model.User{ID: "u1", UserName: "u1"}
			r = r.WithContext(request.WithUser(r.Context(), usr))

			_, err := router.GetCoverArt(w, r)

			Expect(err).ToNot(HaveOccurred())
			got, ok := request.UserFrom(artwork.recvCtx)
			Expect(ok).To(BeTrue(), "the service must see who is asking")
			Expect(got.ID).To(Equal("u1"))
			Expect(got.IsAdmin).To(BeFalse(), "the handler must not elevate")
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

			putRickAstleySong(ctx, ds, "song.mp3", string(lyricsJson))
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
			putRickAstleySong(ctx, ds, "song.mp3", string(lyricsJSON))
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Value).To(Equal("Never gonna let you down\n"))
		})
		It("should return empty subsonic response if the record corresponding to the given artist & title is not found", func() {
			r := newGetRequest("artist=Dheeraj", "title=Rinkiya+Ke+Papa")
			start := int64(0)
			lyricsJSON, err := json.Marshal(model.LyricList{
				{Kind: model.LyricKindMain, Lang: "eng", Line: []model.Line{{Start: &start, Value: "Never gonna give you up"}}},
			})
			Expect(err).ToNot(HaveOccurred())
			putRickAstleySong(ctx, ds, "song.mp3", string(lyricsJSON))
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Artist).To(Equal(""))
			Expect(response.Lyrics.Title).To(Equal(""))
			Expect(response.Lyrics.Value).To(Equal(""))
		})
		It("should return lyric file when finding mediafile with no embedded lyrics but present on filesystem", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			putRickAstleySong(ctx, ds, "tests/fixtures/test.mp3", "")
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
	hash          string
	lastUpdated   time.Time
	placeholder   bool
	err           error
	ctxCancelFunc func()
	recvId        string
	recvSize      int
	recvSquare    bool
	recvCtx       context.Context
}

func (c *fakeArtwork) GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (*artwork.Image, error) {
	c.recvCtx = ctx
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

func useRepoRootAsLibrary(ctx context.Context, ds model.DataStore) {
	GinkgoHelper()
	root, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	persistencetest.SetLibraries(ctx, ds, root)
}

// putRickAstleySong stores "Never Gonna Give You Up" by Rick Astley with ID "1" at path, relative to library 1.
func putRickAstleySong(ctx context.Context, ds model.DataStore, path, lyrics string) {
	GinkgoHelper()
	artist := model.Artist{ID: "ar-rick", Name: "Rick Astley"}
	Expect(ds.Artist().Put(ctx, &artist)).To(Succeed())
	persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{
		ID:           "1",
		Path:         path,
		Artist:       "Rick Astley",
		Title:        "Never Gonna Give You Up",
		Lyrics:       lyrics,
		Participants: model.Participants{model.RoleArtist: {{Artist: artist}}},
	})
}
