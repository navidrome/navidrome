package jellyfin

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AudioMuse info", func() {
	It("lists the sonic endpoints (excluding info) when a provider is present", func() {
		api := &Router{sonic: &fakeSonicEngine{provider: true}}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/AudioMuseAI/info", nil)

		api.audioMuseInfo(w, r)

		Expect(w.Code).To(Equal(200))
		var body audioMuseInfoResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body.Version).To(Equal(consts.Version))
		Expect(body.AvailableEndpoints).To(ConsistOf(
			"GET /AudioMuseAI/similar_tracks",
			"GET /AudioMuseAI/find_path",
		))
	})

	It("returns an empty endpoint list when no provider is loaded", func() {
		api := &Router{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/AudioMuseAI/info", nil)

		api.audioMuseInfo(w, r)

		Expect(w.Code).To(Equal(200))
		var body audioMuseInfoResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body.AvailableEndpoints).To(BeEmpty())
	})
})

type fakeSonicEngine struct {
	provider   bool
	similar    []sonic.SimilarMatch
	similarErr error
	path       []sonic.SimilarMatch
	pathErr    error
	gotID      string
	gotStart   string
	gotEnd     string
	gotCount   int
}

func (f *fakeSonicEngine) HasProvider() bool { return f.provider }

func (f *fakeSonicEngine) GetSonicSimilarTracks(_ context.Context, id string, count int) ([]sonic.SimilarMatch, error) {
	f.gotID, f.gotCount = id, count
	return f.similar, f.similarErr
}

func (f *fakeSonicEngine) FindSonicPath(_ context.Context, startID, endID string, count int) ([]sonic.SimilarMatch, error) {
	f.gotStart, f.gotEnd, f.gotCount = startID, endID, count
	return f.path, f.pathErr
}

func mf(id, artist, title string, lib int) model.MediaFile {
	return model.MediaFile{ID: id, Artist: artist, Title: title, LibraryID: lib}
}

var _ = Describe("AudioMuse similar_tracks", func() {
	var fake *fakeSonicEngine
	var api *Router

	call := func(query string, user model.User) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/AudioMuseAI/similar_tracks?"+query, nil)
		r = r.WithContext(request.WithUser(r.Context(), user))
		invoke(api.audioMuseSimilarTracks, w, r)
		return w
	}

	BeforeEach(func() {
		fake = &fakeSonicEngine{provider: true}
		api = &Router{sonic: fake}
	})

	It("maps matches, decodes the seed id, encodes item ids, copies distance", func() {
		fake.similar = []sonic.SimilarMatch{
			{MediaFile: mf("mf1", "A", "T1", 1), Similarity: 0.3},
			{MediaFile: mf("mf2", "B", "T2", 1), Similarity: 0.5},
		}
		w := call("item_id="+dto.EncodeID("seed")+"&n=5", model.User{IsAdmin: true})

		Expect(w.Code).To(Equal(200))
		Expect(fake.gotID).To(Equal("seed"))
		Expect(fake.gotCount).To(Equal(5))
		var body []audioMuseSimilarTrack
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body).To(HaveLen(2))
		Expect(body[0]).To(Equal(audioMuseSimilarTrack{
			Author: "A", Distance: 0.3, ItemID: dto.EncodeID("mf1"), Title: "T1",
		}))
	})

	It("collapses to one track per artist when eliminate_duplicates defaults on", func() {
		fake.similar = []sonic.SimilarMatch{
			{MediaFile: mf("mf1", "A", "T1", 1), Similarity: 0.3},
			{MediaFile: mf("mf2", "A", "T2", 1), Similarity: 0.5},
		}
		w := call("item_id="+dto.EncodeID("seed"), model.User{IsAdmin: true})
		var body []audioMuseSimilarTrack
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body).To(HaveLen(1))
	})

	It("keeps same-artist tracks when eliminate_duplicates=false", func() {
		fake.similar = []sonic.SimilarMatch{
			{MediaFile: mf("mf1", "A", "T1", 1), Similarity: 0.3},
			{MediaFile: mf("mf2", "A", "T2", 1), Similarity: 0.5},
		}
		w := call("item_id="+dto.EncodeID("seed")+"&eliminate_duplicates=false", model.User{IsAdmin: true})
		var body []audioMuseSimilarTrack
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body).To(HaveLen(2))
	})

	It("filters out tracks in libraries the user cannot access", func() {
		fake.similar = []sonic.SimilarMatch{{MediaFile: mf("mf1", "A", "T1", 2), Similarity: 0.3}}
		w := call("item_id="+dto.EncodeID("seed"), model.User{Libraries: model.Libraries{{ID: 1}}})
		Expect(strings.TrimSpace(w.Body.String())).To(Equal("[]"))
	})

	It("returns an empty array without calling the engine when item_id is missing", func() {
		w := call("n=5", model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(200))
		Expect(strings.TrimSpace(w.Body.String())).To(Equal("[]"))
		Expect(fake.gotID).To(Equal(""))
	})

	It("returns 404 when no sonic provider is loaded", func() {
		fake.provider = false
		w := call("item_id="+dto.EncodeID("seed"), model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(404))
	})
})
