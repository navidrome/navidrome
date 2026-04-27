package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic"
	"github.com/navidrome/navidrome/server/subsonic/responses"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// buildSonicRouter creates a subsonic.Router with a real sonic.Sonic service
// backed by the given provider and the shared e2e DataStore.
func buildSonicRouter(provider sonic.Provider) *subsonic.Router {
	loader := &mockSonicPluginLoader{provider: provider}
	m := matcher.New(ds)
	sonicSvc := sonic.New(ds, loader, m)
	decider := stream.NewTranscodeDecider(ds, noopFFmpeg{})
	return subsonic.New(
		ds,
		noopArtwork{},
		&spyStreamer{},
		noopArchiver{},
		core.NewPlayers(ds),
		noopProvider{},
		nil, // scanner
		events.NoopBroker(),
		playlists.NewPlaylists(ds, core.NewImageUploadService()),
		noopPlayTracker{},
		core.NewShare(ds),
		playback.PlaybackServer(nil),
		metrics.NewNoopInstance(),
		lyrics.NewLyrics(nil),
		decider,
		sonicSvc,
	)
}

// doSonicReq makes a request through a sonic-enabled router and returns the parsed response.
func doSonicReq(sonicRouter *subsonic.Router, endpoint string, params ...string) *responses.Subsonic {
	w := httptest.NewRecorder()
	r := buildReq(adminUser, endpoint, params...)
	sonicRouter.ServeHTTP(w, r)
	return parseJSONResponse(w)
}

// doSonicRawReq makes a request through a sonic-enabled router and returns the raw recorder.
func doSonicRawReq(sonicRouter *subsonic.Router, endpoint string, params ...string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := buildReq(adminUser, endpoint, params...)
	sonicRouter.ServeHTTP(w, r)
	return w
}

var _ = Describe("Sonic Similarity Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Context("without sonic similarity plugin", func() {
		Describe("getSonicSimilarTracks", func() {
			It("returns 404 when no sonic similarity plugin is available", func() {
				w := doRawReq("getSonicSimilarTracks", "id", "any-song-id")
				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})

		Describe("findSonicPath", func() {
			It("returns 404 when no sonic similarity plugin is available", func() {
				w := doRawReq("findSonicPath", "startSongId", "any-song-id", "endSongId", "another-song-id")
				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})
	})

	Context("with sonic similarity plugin", func() {
		var (
			sonicRouter  *subsonic.Router
			comeTogether model.MediaFile
			something    model.MediaFile
		)

		BeforeEach(func() {
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"title": "Come Together"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			comeTogether = songs[0]

			songs, err = ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"title": "Something"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			something = songs[0]

			provider := &mockSonicProvider{
				similarIDs: []string{something.ID, comeTogether.ID},
				pathIDs:    []string{comeTogether.ID, something.ID},
			}
			sonicRouter = buildSonicRouter(provider)
		})

		Describe("getSonicSimilarTracks", func() {
			It("returns similar tracks with similarity scores", func() {
				resp := doSonicReq(sonicRouter, "getSonicSimilarTracks", "id", comeTogether.ID)

				Expect(resp.Status).To(Equal(responses.StatusOK))
				matches := *resp.SonicMatches
				Expect(matches).To(HaveLen(2))
				Expect(matches[0].Entry.Title).To(Equal("Something"))
				Expect(matches[0].Similarity).To(Equal(1.0))
				Expect(matches[1].Entry.Title).To(Equal("Come Together"))
				Expect(matches[1].Similarity).To(Equal(0.9))
			})

			It("respects the count parameter", func() {
				resp := doSonicReq(sonicRouter, "getSonicSimilarTracks", "id", comeTogether.ID, "count", "1")

				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(*resp.SonicMatches).To(HaveLen(1))
			})

			It("returns an error for a missing id parameter", func() {
				resp := doSonicReq(sonicRouter, "getSonicSimilarTracks")

				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})

			It("returns an error for a non-existent song ID", func() {
				resp := doSonicReq(sonicRouter, "getSonicSimilarTracks", "id", "non-existent-id")

				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})

			It("returns correct JSON structure", func() {
				w := doSonicRawReq(sonicRouter, "getSonicSimilarTracks", "id", comeTogether.ID)
				Expect(w.Code).To(Equal(http.StatusOK))

				var wrapper responses.JsonWrapper
				Expect(json.Unmarshal(w.Body.Bytes(), &wrapper)).To(Succeed())
				matches := *wrapper.Subsonic.SonicMatches
				Expect(matches).To(HaveLen(2))
				Expect(matches[0].Similarity).To(BeNumerically(">", 0))
				Expect(matches[0].Entry.Id).ToNot(BeEmpty())
			})
		})

		Describe("findSonicPath", func() {
			It("returns a path between two tracks with similarity scores", func() {
				resp := doSonicReq(sonicRouter, "findSonicPath",
					"startSongId", comeTogether.ID,
					"endSongId", something.ID,
				)

				Expect(resp.Status).To(Equal(responses.StatusOK))
				matches := *resp.SonicMatches
				Expect(matches).To(HaveLen(2))
				Expect(matches[0].Entry.Title).To(Equal("Come Together"))
				Expect(matches[0].Similarity).To(Equal(1.0))
				Expect(matches[1].Entry.Title).To(Equal("Something"))
				Expect(matches[1].Similarity).To(Equal(0.95))
			})

			It("returns an error for a missing startSongId parameter", func() {
				resp := doSonicReq(sonicRouter, "findSonicPath", "endSongId", something.ID)

				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})

			It("returns an error for a missing endSongId parameter", func() {
				resp := doSonicReq(sonicRouter, "findSonicPath", "startSongId", comeTogether.ID)

				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})

			It("returns an error for a non-existent start song ID", func() {
				resp := doSonicReq(sonicRouter, "findSonicPath",
					"startSongId", "non-existent-id",
					"endSongId", something.ID,
				)

				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})

			It("returns an error for a non-existent end song ID", func() {
				resp := doSonicReq(sonicRouter, "findSonicPath",
					"startSongId", comeTogether.ID,
					"endSongId", "non-existent-id",
				)

				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})
		})
	})
})

// mockSonicProvider returns results using IDs from the real test library,
// so that the matcher can resolve them back to actual MediaFiles.
type mockSonicProvider struct {
	similarIDs []string
	pathIDs    []string
}

func (m *mockSonicProvider) GetSonicSimilarTracks(_ context.Context, mf *model.MediaFile, count int) ([]sonic.SimilarResult, error) {
	var results []sonic.SimilarResult
	for i, id := range m.similarIDs {
		if i >= count {
			break
		}
		results = append(results, sonic.SimilarResult{
			Song:       agents.Song{ID: id},
			Similarity: 1.0 - float64(i)*0.1,
		})
	}
	return results, nil
}

func (m *mockSonicProvider) FindSonicPath(_ context.Context, startMf, endMf *model.MediaFile, count int) ([]sonic.SimilarResult, error) {
	var results []sonic.SimilarResult
	for i, id := range m.pathIDs {
		if i >= count {
			break
		}
		results = append(results, sonic.SimilarResult{
			Song:       agents.Song{ID: id},
			Similarity: 1.0 - float64(i)*0.05,
		})
	}
	return results, nil
}

type mockSonicPluginLoader struct {
	provider sonic.Provider
}

func (m *mockSonicPluginLoader) PluginNames(capability string) []string {
	if capability == "SonicSimilarity" && m.provider != nil {
		return []string{"mock-sonic"}
	}
	return nil
}

func (m *mockSonicPluginLoader) LoadSonicSimilarity(_ string) (sonic.Provider, bool) {
	if m.provider != nil {
		return m.provider, true
	}
	return nil, false
}
