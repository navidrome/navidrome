package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Streaming", func() {
	BeforeEach(func() { setupTestDB() })

	Describe("GET /Audio/{id}/stream", func() {
		It("streams the requested track", func() {
			id := songID("Come Together")
			w := get("/Audio/" + enc(id) + "/stream")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("fake audio data"))
			Expect(streamerSpy.LastMediaFile.ID).To(Equal(id))
		})

		It("streams via the /universal endpoint", func() {
			id := songID("So What")
			Expect(get("/Audio/" + enc(id) + "/universal").Code).To(Equal(http.StatusOK))
			Expect(streamerSpy.LastMediaFile.ID).To(Equal(id))
		})

		It("serves the stream.{container} path form", func() {
			id := songID("Help!")
			Expect(get("/Audio/" + enc(id) + "/stream.mp3").Code).To(Equal(http.StatusOK))
			Expect(streamerSpy.LastMediaFile.ID).To(Equal(id))
		})

		It("forces raw format when static=true", func() {
			// With ffmpeg unavailable the decider direct-plays regardless, but static=true must
			// never resolve to a transcode.
			id := songID("Help!")
			get("/Audio/" + enc(id) + "/stream?static=true")
			Expect(streamerSpy.LastRequest.Format).To(Equal("raw"))
		})

		It("returns 404 for an unknown track", func() {
			Expect(get("/Audio/" + enc("nope") + "/stream").Code).To(Equal(http.StatusNotFound))
		})

		It("streams when authenticated only by a bare Authorization token (Jellify native player)", func() {
			id := songID("Come Together")
			w := getWithBareToken("/Audio/" + enc(id) + "/stream?playSessionId=x&static=true")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(streamerSpy.LastMediaFile.ID).To(Equal(id))
		})
	})

	Describe("direct-file endpoints", func() {
		It("serves /Items/{id}/File as direct play (raw)", func() {
			id := songID("Something")
			w := get("/Items/" + enc(id) + "/File")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(streamerSpy.LastRequest.Format).To(Equal("raw"))
		})

		It("serves /Items/{id}/Download", func() {
			id := songID("Something")
			Expect(get("/Items/" + enc(id) + "/Download").Code).To(Equal(http.StatusOK))
		})
	})

	Describe("PlaybackInfo", func() {
		It("returns a single direct-play MediaSource via GET", func() {
			id := songID("So What")
			var info dto.PlaybackInfoResponse
			parseInto(get("/Items/"+enc(id)+"/PlaybackInfo"), &info)
			Expect(info.MediaSources).To(HaveLen(1))
			Expect(info.MediaSources[0].Id).ToNot(BeEmpty())
			Expect(info.PlaySessionId).ToNot(BeEmpty())
		})

		It("returns a MediaSource via POST", func() {
			id := songID("So What")
			var info dto.PlaybackInfoResponse
			parseInto(post("/Items/"+enc(id)+"/PlaybackInfo", "{}"), &info)
			Expect(info.MediaSources).To(HaveLen(1))
		})
	})
})
