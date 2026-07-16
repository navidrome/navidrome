package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeLyricsService returns canned lyrics per media-file ID and counts calls.
type fakeLyricsService struct {
	lyrics      map[string]model.LyricList
	err         error
	calls       int
	hadDeadline bool
}

func (f *fakeLyricsService) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	f.calls++
	_, f.hadDeadline = ctx.Deadline()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.lyrics[mf.ID], nil
}

func (f *fakeLyricsService) GetLyricsByArtistTitle(context.Context, string, string) (model.LyricList, error) {
	return nil, nil
}

func p(ms int64) *int64 { return &ms }

func newTestLyricsCache() cache.SimpleCache[string, model.LyricList] {
	return cache.NewSimpleCache[string, model.LyricList](cache.Options{SizeLimit: 1000})
}

var _ = Describe("getLyrics", func() {
	var api *Router
	var ds *tests.MockDataStore
	var fake *fakeLyricsService

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "s1", Title: "Song", LibraryID: 1},
			{ID: "s2", Title: "Silent Song", LibraryID: 1},
		})
		fake = &fakeLyricsService{lyrics: map[string]model.LyricList{}}
		api = &Router{
			ds:          ds,
			lyrics:      fake,
			lyricsCache: newTestLyricsCache(),
		}
	})

	doRequest := func(id string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		ctx := request.WithUser(context.Background(), model.User{ID: "u1", Libraries: model.Libraries{{ID: 1}}})
		// Clients send hex-encoded ids (matching real traffic and the other handler tests).
		enc := dto.EncodeID(id)
		r := httptest.NewRequest("GET", "/Audio/"+enc+"/Lyrics", nil).WithContext(ctx)
		r = withChiURLParam(r, "itemId", enc)
		invoke(api.getLyrics, w, r)
		return w
	}

	It("returns 200 with a LyricDto for a track with synced lyrics", func() {
		fake.lyrics["s1"] = model.LyricList{
			{Kind: "main", Synced: true, Line: []model.Line{{Start: p(1000), Value: "hello"}}},
		}
		w := doRequest("s1")

		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.LyricDto
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Lyrics).To(HaveLen(1))
		Expect(res.Lyrics[0].Text).To(Equal("hello"))
		Expect(res.Lyrics[0].Start).ToNot(BeNil())
		Expect(*res.Lyrics[0].Start).To(Equal(int64(10000000)))
	})

	It("serves the main-kind lyric when a translation is also present", func() {
		fake.lyrics["s1"] = model.LyricList{
			{Kind: "translation", Synced: true, Line: []model.Line{{Start: p(1000), Value: "bonjour"}}},
			{Kind: "main", Synced: true, Line: []model.Line{{Start: p(1000), Value: "hello"}}},
		}
		w := doRequest("s1")

		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.LyricDto
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Lyrics).To(HaveLen(1))
		Expect(res.Lyrics[0].Text).To(Equal("hello"))
	})

	It("returns 404 when the service returns no lyrics", func() {
		w := doRequest("s2")
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 404 when the main lyric has no lines", func() {
		fake.lyrics["s1"] = model.LyricList{{Kind: "main", Lang: "eng"}}
		w := doRequest("s1")
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 404 for an unknown item id", func() {
		w := doRequest("unknown")
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("caches results so a second request doesn't re-invoke the service", func() {
		fake.lyrics["s1"] = model.LyricList{
			{Kind: "main", Synced: true, Line: []model.Line{{Start: p(1000), Value: "hello"}}},
		}
		Expect(doRequest("s1").Code).To(Equal(http.StatusOK))
		Expect(doRequest("s1").Code).To(Equal(http.StatusOK))
		Expect(fake.calls).To(Equal(1))
	})

	It("caches empty results too", func() {
		Expect(doRequest("s2").Code).To(Equal(http.StatusNotFound))
		Expect(doRequest("s2").Code).To(Equal(http.StatusNotFound))
		Expect(fake.calls).To(Equal(1))
	})

	It("completes and caches the fetch even when the request context is cancelled", func() {
		fake.lyrics["s1"] = model.LyricList{
			{Kind: "main", Synced: true, Line: []model.Line{{Start: p(1000), Value: "hello"}}},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		list := api.cachedLyrics(ctx, &model.MediaFile{ID: "s1"})
		Expect(list).ToNot(BeEmpty())
		Expect(doRequest("s1").Code).To(Equal(http.StatusOK))
		Expect(fake.calls).To(Equal(1))
	})

	It("bounds the detached fetch with a timeout", func() {
		Expect(doRequest("s2").Code).To(Equal(http.StatusNotFound))
		Expect(fake.hadDeadline).To(BeTrue())
	})
})
