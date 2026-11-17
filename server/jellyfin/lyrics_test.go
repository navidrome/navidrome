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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeLyricsService returns canned lyrics per media-file ID.
type fakeLyricsService struct {
	lyrics   map[string]model.LyricList
	err      error
	panicVal any
	blockCh  <-chan struct{}
	doneCh   chan<- struct{}
}

func (f *fakeLyricsService) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	if f.doneCh != nil {
		defer close(f.doneCh)
	}
	if f.blockCh != nil {
		select {
		case <-f.blockCh:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.panicVal != nil {
		panic(f.panicVal) //nolint:forbidigo
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.lyrics[mf.ID], nil
}

func (f *fakeLyricsService) GetLyricsByArtistTitle(context.Context, string, string) (model.LyricList, error) {
	return nil, nil
}

var _ = Describe("getLyrics", func() {
	var ctx context.Context
	var api *Router
	var ds *tests.MockDataStore
	var fake *fakeLyricsService

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = &tests.MockDataStore{}
		ds.MediaFile().(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: testID("s1"), Title: "Song", LibraryID: 1},
			{ID: testID("s2"), Title: "Silent Song", LibraryID: 1},
		})
		fake = &fakeLyricsService{lyrics: map[string]model.LyricList{}}
		api = &Router{
			ds:     ds,
			lyrics: fake,
		}
	})

	doRequest := func(id string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		// Clients send hex-encoded ids (matching real traffic and the other handler tests).
		enc := dto.EncodeID(id)
		r := httptest.NewRequest("GET", "/Audio/"+enc+"/Lyrics", nil).WithContext(request.WithUser(ctx, model.User{ID: testID("u1"), Libraries: model.Libraries{{ID: 1}}}))
		r = withChiURLParam(r, "itemId", enc)
		invoke(api.getLyrics, w, r)
		return w
	}

	It("returns 200 with a LyricDto for a track with synced lyrics", func() {
		fake.lyrics[testID("s1")] = model.LyricList{
			{Kind: "main", Synced: true, Line: []model.Line{{Start: new(int64(1000)), Value: "hello"}}},
		}
		w := doRequest(testID("s1"))

		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.LyricDto
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Lyrics).To(HaveLen(1))
		Expect(res.Lyrics[0].Text).To(Equal("hello"))
		Expect(res.Lyrics[0].Start).ToNot(BeNil())
		Expect(*res.Lyrics[0].Start).To(Equal(int64(10000000)))
	})

	It("serves the main-kind lyric when a translation is also present", func() {
		fake.lyrics[testID("s1")] = model.LyricList{
			{Kind: "translation", Synced: true, Line: []model.Line{{Start: new(int64(1000)), Value: "bonjour"}}},
			{Kind: "main", Synced: true, Line: []model.Line{{Start: new(int64(1000)), Value: "hello"}}},
		}
		w := doRequest(testID("s1"))

		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.LyricDto
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Lyrics).To(HaveLen(1))
		Expect(res.Lyrics[0].Text).To(Equal("hello"))
	})

	It("returns 404 when the service returns no lyrics", func() {
		w := doRequest(testID("s2"))
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 404 when the main lyric has no lines", func() {
		fake.lyrics[testID("s1")] = model.LyricList{{Kind: "main", Lang: "eng"}}
		w := doRequest(testID("s1"))
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 404 for an unknown item id", func() {
		w := doRequest("unknown")
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})
