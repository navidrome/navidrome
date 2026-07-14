package e2e

import (
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AudioMuse endpoints", func() {
	BeforeEach(func() { setupTestDB() })

	Describe("GET /AudioMuseAI/info", func() {
		It("returns version and available endpoints", func() {
			var body struct {
				Version            string   `json:"Version"`
				AvailableEndpoints []string `json:"AvailableEndpoints"`
			}
			parseInto(get("/AudioMuseAI/info"), &body)
			Expect(body.Version).To(Equal(consts.Version))
			Expect(body.AvailableEndpoints).To(ConsistOf(
				"GET /AudioMuseAI/find_path",
				"GET /AudioMuseAI/health",
				"GET /AudioMuseAI/similar_tracks",
			))
		})

		It("requires authentication", func() {
			Expect(rawReq("GET", "/AudioMuseAI/info", "").Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("GET /AudioMuseAI/health", func() {
		It("returns 200 with an empty body when a provider is loaded", func() {
			w := get("/AudioMuseAI/health")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.Len()).To(Equal(0))
		})

		It("requires authentication", func() {
			Expect(rawReq("GET", "/AudioMuseAI/health", "").Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("GET /AudioMuseAI/similar_tracks", func() {
		It("maps provider results to seeded tracks, encoding item ids", func() {
			sonicProviderFake.similar = []sonic.SimilarResult{
				{Song: songAgent("Something"), Similarity: 0.3},
				{Song: songAgent("So What"), Similarity: 0.5},
			}
			var body []struct {
				Author   string  `json:"author"`
				Distance float64 `json:"distance"`
				ItemID   string  `json:"item_id"`
				Title    string  `json:"title"`
			}
			parseInto(get("/AudioMuseAI/similar_tracks?item_id="+enc(songID("Come Together"))+"&n=10"), &body)
			Expect(body).To(HaveLen(2))
			Expect([]string{body[0].Title, body[1].Title}).To(ConsistOf("Something", "So What"))
			Expect(dto.DecodeID(body[0].ItemID)).To(Equal(songID(body[0].Title)))
		})

		It("collapses to one track per artist by default", func() {
			sonicProviderFake.similar = []sonic.SimilarResult{
				{Song: songAgent("Something"), Similarity: 0.3},
				{Song: songAgent("Come Together"), Similarity: 0.5},
			}
			var body []map[string]any
			parseInto(get("/AudioMuseAI/similar_tracks?item_id="+enc(songID("Help!"))), &body)
			Expect(body).To(HaveLen(1)) // both similar tracks are by The Beatles
		})

		It("returns an empty array (not null) when there are no results", func() {
			sonicProviderFake.similar = nil
			w := get("/AudioMuseAI/similar_tracks?item_id=" + enc(songID("Come Together")))
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(strings.TrimSpace(w.Body.String())).To(Equal("[]"))
		})

		It("requires authentication", func() {
			Expect(rawReq("GET", "/AudioMuseAI/similar_tracks?item_id=x", "").Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("GET /AudioMuseAI/find_path", func() {
		It("returns 400 with the exact message when a required id is missing", func() {
			w := get("/AudioMuseAI/find_path?start_song_id=" + enc(songID("Something")))
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(strings.TrimSpace(w.Body.String())).To(Equal("start_song_id and end_song_id are required."))
		})

		It("returns the path and summed total_distance", func() {
			sonicProviderFake.path = []sonic.SimilarResult{
				{Song: songAgent("Come Together"), Similarity: 1.5},
				{Song: songAgent("So What"), Similarity: 2.0},
			}
			var body struct {
				Path []struct {
					ItemID string `json:"item_id"`
					Title  string `json:"title"`
				} `json:"path"`
				TotalDistance float64 `json:"total_distance"`
			}
			parseInto(get("/AudioMuseAI/find_path?start_song_id="+enc(songID("Something"))+"&end_song_id="+enc(songID("So What"))+"&max_steps=10"), &body)
			Expect(body.Path).To(HaveLen(2))
			Expect(body.TotalDistance).To(Equal(3.5))
		})
	})
})
