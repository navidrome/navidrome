package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("awaitSimilar", func() {
	var api *Router
	ctxFor := func(userID string) context.Context {
		return request.WithUser(context.Background(), model.User{ID: userID})
	}
	shortenWait := func() {
		old := similarWait
		similarWait = 20 * time.Millisecond
		DeferCleanup(func() { similarWait = old })
	}

	BeforeEach(func() {
		api = &Router{}
	})

	It("returns the fetch result when it completes within the wait", func() {
		res := api.awaitSimilar(ctxFor(testID("u1")), "id1", 20, func(context.Context) dto.QueryResult {
			return result([]dto.BaseItemDto{{Name: "fast"}}, 1, 0)
		})
		Expect(res.Items).To(HaveLen(1))
		Expect(res.Items[0].Name).To(Equal("fast"))
	})

	It("returns an empty result when the fetch exceeds the wait", func() {
		shortenWait()
		release := make(chan struct{})
		DeferCleanup(func() { close(release) })
		res := api.awaitSimilar(ctxFor(testID("u1")), "id2", 20, func(context.Context) dto.QueryResult {
			<-release // hung provider; would finish caching in the background
			return result([]dto.BaseItemDto{{Name: "late"}}, 1, 0)
		})
		Expect(res.Items).To(BeEmpty())
		Expect(res.TotalRecordCount).To(Equal(0))
	})

	It("dedupes requests into the in-flight fetch", func() {
		shortenWait()
		var calls atomic.Int32
		release := make(chan struct{})
		fetch := func(context.Context) dto.QueryResult {
			calls.Add(1)
			<-release
			return result(nil, 0, 0)
		}
		// Both calls time out, but the flight can't complete before release closes, so the
		// second call must join it rather than start a new fetch.
		api.awaitSimilar(ctxFor(testID("u1")), "id3", 20, fetch)
		api.awaitSimilar(ctxFor(testID("u1")), "id3", 20, fetch)
		close(release)
		Eventually(calls.Load).Should(Equal(int32(1)))
		Consistently(calls.Load, "50ms").Should(Equal(int32(1)))
	})

	It("does not share fetches across users (items embed the user's annotations)", func() {
		var calls atomic.Int32
		fetch := func(context.Context) dto.QueryResult {
			calls.Add(1)
			return result(nil, 0, 0)
		}
		api.awaitSimilar(ctxFor(testID("u1")), "id4", 20, fetch)
		api.awaitSimilar(ctxFor(testID("u2")), "id4", 20, fetch)
		Expect(calls.Load()).To(Equal(int32(2)))
	})

	It("hands the fetch a deadline-bounded background context", func() {
		var deadline time.Time
		var hasDeadline bool
		api.awaitSimilar(ctxFor(testID("u1")), "id5", 20, func(ctx context.Context) dto.QueryResult {
			deadline, hasDeadline = ctx.Deadline()
			return result(nil, 0, 0)
		})
		Expect(hasDeadline).To(BeTrue(), "background fetch must not be able to run forever")
		Expect(time.Until(deadline)).To(BeNumerically("<=", similarFetchTimeout))
	})
})

// blockingProvider hangs SimilarSongs until release is closed, simulating a slow/unreachable agent.
type blockingProvider struct {
	external.Provider
	release chan struct{}
}

func (p *blockingProvider) SimilarSongs(context.Context, string, int) (model.MediaFiles, error) {
	<-p.release
	return nil, nil
}

// fakeSimilarProvider returns up to count of its canned songs, like a real agent honoring the limit.
type fakeSimilarProvider struct {
	external.Provider
	songs model.MediaFiles
}

func (p *fakeSimilarProvider) SimilarSongs(_ context.Context, _ string, count int) (model.MediaFiles, error) {
	return p.songs[:min(count, len(p.songs))], nil
}

var _ = Describe("getInstantMix", func() {
	It("returns the seed track even when the provider fetch exceeds the wait", func() {
		old := similarWait
		similarWait = 20 * time.Millisecond
		DeferCleanup(func() { similarWait = old })

		ds := &tests.MockDataStore{}
		ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: testID("s1"), Title: "Seed Song", LibraryID: 1},
		})
		release := make(chan struct{})
		DeferCleanup(func() { close(release) })
		api := &Router{ds: ds, provider: &blockingProvider{release: release}}

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items/"+dto.EncodeID(testID("s1"))+"/InstantMix", nil).
			WithContext(request.WithUser(context.Background(), model.User{ID: testID("u1"), Libraries: model.Libraries{{ID: 1}}}))
		r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
		api.getInstantMix(w, r)

		var res dto.QueryResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Items).To(HaveLen(1))
		Expect(res.Items[0].Name).To(Equal("Seed Song"))
	})

	// Finamp's Radio Mix asks for limit=250. Clamping that to the Similar ceiling (100) truncated the
	// queue, so InstantMix gets its own, higher ceiling.
	It("honors a mix-sized limit above the Similar ceiling", func() {
		const want = 250
		songs := model.MediaFiles{{ID: testID("s1"), Title: "Seed Song", LibraryID: 1}}
		for i := range want + 50 { // more than requested, so only the limit bounds the result
			songs = append(songs, model.MediaFile{ID: testID(fmt.Sprintf("t%d", i)), Title: fmt.Sprintf("Track %d", i), LibraryID: 1})
		}
		ds := &tests.MockDataStore{}
		ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(songs)
		api := &Router{ds: ds, provider: &fakeSimilarProvider{songs: songs[1:]}}

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items/"+dto.EncodeID(testID("s1"))+"/InstantMix?limit="+strconv.Itoa(want), nil).
			WithContext(request.WithUser(context.Background(), model.User{ID: testID("u1"), Libraries: model.Libraries{{ID: 1}}}))
		r = withChiURLParam(r, "itemId", dto.EncodeID(testID("s1")))
		api.getInstantMix(w, r)

		var res dto.QueryResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Items).To(HaveLen(want), "a Radio Mix-sized request must not be truncated to the Similar ceiling")
		Expect(res.Items[0].Name).To(Equal("Seed Song"), "the seed must still lead the mix")
	})

	It("returns a mix for a genre id, which GetEntityByID can't resolve", func() {
		ds := &tests.MockDataStore{}
		songs := model.MediaFiles{
			{ID: testID("m1"), Title: "Track 1", LibraryID: 1},
			{ID: testID("m2"), Title: "Track 2", LibraryID: 1},
		}
		api := &Router{ds: ds, provider: &fakeSimilarProvider{songs: songs}}

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items/"+dto.EncodeID(testID("g1"))+"/InstantMix", nil).
			WithContext(request.WithUser(context.Background(), model.User{ID: testID("u1"), Libraries: model.Libraries{{ID: 1}}}))
		r = withChiURLParam(r, "itemId", dto.EncodeID(testID("g1")))
		api.getInstantMix(w, r)

		Expect(w.Code).To(Equal(200))
		var res dto.QueryResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Items).To(HaveLen(2))
	})
})

var _ = Describe("getSimilarAlbums", func() {
	It("does not return the seed album as its own similar album", func() {
		// With no external agent the provider falls back to the album's own tracks, which map
		// straight back to the requested album.
		ds := &tests.MockDataStore{}
		ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: testID("al-1"), Name: "Seed Album", LibraryID: 1},
		})
		api := &Router{ds: ds, provider: &fakeSimilarProvider{
			songs: model.MediaFiles{{ID: testID("m1"), AlbumID: testID("al-1"), LibraryID: 1}},
		}}

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Albums/"+dto.EncodeID(testID("al-1"))+"/Similar?limit=10", nil).
			WithContext(request.WithUser(context.Background(), model.User{ID: testID("u1"), Libraries: model.Libraries{{ID: 1}}}))
		r = withChiURLParam(r, "itemId", dto.EncodeID(testID("al-1")))
		api.getSimilarAlbums(w, r)

		Expect(w.Code).To(Equal(200))
		var res dto.QueryResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Items).To(BeEmpty())
	})

	It("returns albums derived from the provider's similar songs", func() {
		ds := &tests.MockDataStore{}
		ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: testID("al-2"), Name: "Other", LibraryID: 1},
		})
		api := &Router{ds: ds, provider: &fakeSimilarProvider{
			songs: model.MediaFiles{{ID: testID("m1"), AlbumID: testID("al-2"), LibraryID: 1}},
		}}

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Albums/"+dto.EncodeID(testID("al-1"))+"/Similar?limit=10", nil).
			WithContext(request.WithUser(context.Background(), model.User{ID: testID("u1"), Libraries: model.Libraries{{ID: 1}}}))
		r = withChiURLParam(r, "itemId", dto.EncodeID(testID("al-1")))
		api.getSimilarAlbums(w, r)

		Expect(w.Code).To(Equal(200))
		var res dto.QueryResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Items).To(HaveLen(1))
		Expect(res.Items[0].Name).To(Equal("Other"))
	})
})
