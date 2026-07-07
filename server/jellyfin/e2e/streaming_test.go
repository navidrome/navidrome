package e2e

import (
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/consts"
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
	})

	Describe("GET /Audio/{id}/main.m3u8 (Finamp transcoding mode)", func() {
		It("returns a VOD playlist whose segment streams through the transcode pipeline", func() {
			id := songID("Come Together")
			w := get("/Audio/" + enc(id) + "/main.m3u8?audioCodec=aac&audioBitRate=320000")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/vnd.apple.mpegurl"))
			body := w.Body.String()
			Expect(body).To(HavePrefix("#EXTM3U\n"))
			Expect(body).To(HaveSuffix("#EXT-X-ENDLIST\n"))

			// Fetch the advertised segment like an HLS player would.
			var segment string
			for _, line := range strings.Split(body, "\n") {
				if line != "" && !strings.HasPrefix(line, "#") {
					segment = line
				}
			}
			Expect(segment).To(HavePrefix("stream.aac?"))
			Expect(get("/Audio/" + enc(id) + "/" + segment).Code).To(Equal(http.StatusOK))
			Expect(streamerSpy.LastMediaFile.ID).To(Equal(id))
			Expect(streamerSpy.LastRequest.Format).To(Equal("aac"))
			Expect(streamerSpy.LastRequest.BitRate).To(Equal(320))
		})

		It("is reachable with Jellyfin's case-insensitive routing", func() {
			id := songID("Come Together")
			Expect(get("/audio/" + enc(id) + "/Main.m3u8").Code).To(Equal(http.StatusOK))
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

		It("embeds a self-authenticating TranscodingUrl (for native players that omit auth headers)", func() {
			id := songID("So What")
			var info dto.PlaybackInfoResponse
			parseInto(get("/Items/"+enc(id)+"/PlaybackInfo"), &info)
			streamURL := info.MediaSources[0].TranscodingUrl
			// The URL includes the /jellyfin mount prefix so a client resolving it as an absolute
			// host path hits the mounted router.
			Expect(streamURL).To(HavePrefix(consts.URLPathJellyfinAPI + "/Audio/" + enc(id) + "/universal"))
			Expect(streamURL).To(ContainSubstring("api_key="))
			// The embedded api_key alone must authenticate the stream — no auth header sent. The e2e
			// router is mounted at the root, so strip the /jellyfin prefix before replaying.
			replayURL := strings.TrimPrefix(streamURL, consts.URLPathJellyfinAPI)
			w := rawReq("GET", replayURL, "")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(streamerSpy.LastMediaFile.ID).To(Equal(id))
		})
	})
})
