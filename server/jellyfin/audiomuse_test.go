package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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
			"GET /AudioMuseAI/find_path",
			"GET /AudioMuseAI/health",
			"GET /AudioMuseAI/similar_tracks",
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
		Expect(w.Body.String()).To(ContainSubstring(`"AvailableEndpoints":[]`))
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

var _ = Describe("AudioMuse health", func() {
	It("returns 200 with an empty body when a provider is loaded", func() {
		api := &Router{sonic: &fakeSonicEngine{provider: true}}
		w := audioMuseGet(api.audioMuseHealth, "/AudioMuseAI/health", "", model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(200))
		Expect(w.Body.Len()).To(Equal(0))
	})

	It("returns 404 when no provider is loaded", func() {
		api := &Router{}
		w := audioMuseGet(api.audioMuseHealth, "/AudioMuseAI/health", "", model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(404))
	})
})

// audioMuseGet drives a GET through normalizeQueryKeys as the given user, mirroring a real request.
func audioMuseGet(handler http.HandlerFunc, path, query string, user model.User) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", path+"?"+query, nil)
	r = r.WithContext(request.WithUser(r.Context(), user))
	invoke(handler, w, r)
	return w
}

var _ = Describe("AudioMuse similar_tracks", func() {
	var fake *fakeSonicEngine
	var api *Router

	call := func(query string, user model.User) *httptest.ResponseRecorder {
		return audioMuseGet(api.audioMuseSimilarTracks, "/AudioMuseAI/similar_tracks", query, user)
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

	It("returns an empty array when the engine errors", func() {
		fake.similarErr = errors.New("boom")
		fake.similar = []sonic.SimilarMatch{{MediaFile: mf("mf1", "A", "T1", 1), Similarity: 0.3}}
		w := call("item_id="+dto.EncodeID("seed"), model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(200))
		Expect(strings.TrimSpace(w.Body.String())).To(Equal("[]"))
	})
})

var _ = Describe("AudioMuse find_path", func() {
	var fake *fakeSonicEngine
	var api *Router

	call := func(query string, user model.User) *httptest.ResponseRecorder {
		return audioMuseGet(api.audioMuseFindPath, "/AudioMuseAI/find_path", query, user)
	}

	BeforeEach(func() {
		fake = &fakeSonicEngine{provider: true}
		api = &Router{sonic: fake}
	})

	It("returns 400 with the exact message when start_song_id is missing", func() {
		w := call("end_song_id="+dto.EncodeID("e"), model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(400))
		Expect(strings.TrimSpace(w.Body.String())).To(Equal("start_song_id and end_song_id are required."))
	})

	It("returns 400 when end_song_id is missing", func() {
		w := call("start_song_id="+dto.EncodeID("s"), model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(400))
	})

	It("maps the path, decodes ids, sums total_distance, fills tempo from BPM", func() {
		bpm := 120
		withBPM := mf("mf1", "A", "T1", 1)
		withBPM.BPM = &bpm
		fake.path = []sonic.SimilarMatch{
			{MediaFile: withBPM, Similarity: 1.5},
			{MediaFile: mf("mf2", "B", "T2", 1), Similarity: 2.0},
		}
		w := call("start_song_id="+dto.EncodeID("s")+"&end_song_id="+dto.EncodeID("e")+"&max_steps=10", model.User{IsAdmin: true})

		Expect(w.Code).To(Equal(200))
		Expect(fake.gotStart).To(Equal("s"))
		Expect(fake.gotEnd).To(Equal("e"))
		Expect(fake.gotCount).To(Equal(10))
		var body audioMusePathResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body.Path).To(HaveLen(2))
		Expect(body.TotalDistance).To(Equal(3.5))
		Expect(body.Path[0].ItemID).To(Equal(dto.EncodeID("mf1")))
		Expect(*body.Path[0].Tempo).To(Equal(120.0))
		Expect(body.Path[1].Tempo).To(BeNil())
	})

	It("returns 404 when no sonic provider is loaded", func() {
		fake.provider = false
		w := call("start_song_id="+dto.EncodeID("s")+"&end_song_id="+dto.EncodeID("e"), model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(404))
	})

	It("returns an empty path object when the engine errors", func() {
		fake.pathErr = errors.New("boom")
		fake.path = []sonic.SimilarMatch{{MediaFile: mf("mf1", "A", "T1", 1), Similarity: 1.0}}
		w := call("start_song_id="+dto.EncodeID("s")+"&end_song_id="+dto.EncodeID("e"), model.User{IsAdmin: true})
		Expect(w.Code).To(Equal(200))
		var body audioMusePathResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body.Path).To(BeEmpty())
		Expect(body.TotalDistance).To(Equal(0.0))
	})
})
