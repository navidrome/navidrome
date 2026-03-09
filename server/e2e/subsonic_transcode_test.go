package e2e

import (
	"net/http"
	"time"

	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Client profile JSON bodies for getTranscodeDecision requests.
// All bitrate values are in bps (per OpenSubsonic spec).
const (
	// mp3OnlyClient can direct-play mp3 and transcode to mp3
	mp3OnlyClient = `{
		"name": "test-mp3-only",
		"directPlayProfiles": [
			{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["http"]}
		],
		"transcodingProfiles": [
			{"container": "mp3", "audioCodec": "mp3", "protocol": "http"}
		]
	}`

	// flacAndMp3Client can direct-play flac and mp3, transcode to mp3
	flacAndMp3Client = `{
		"name": "test-flac-mp3",
		"directPlayProfiles": [
			{"containers": ["flac"], "audioCodecs": ["flac"], "protocols": ["http"]},
			{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["http"]}
		],
		"transcodingProfiles": [
			{"container": "mp3", "audioCodec": "mp3", "protocol": "http"}
		]
	}`

	// universalClient can direct-play most formats
	universalClient = `{
		"name": "test-universal",
		"directPlayProfiles": [
			{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["http"]},
			{"containers": ["flac"], "audioCodecs": ["flac"], "protocols": ["http"]},
			{"containers": ["m4a"], "audioCodecs": ["alac", "aac"], "protocols": ["http"]},
			{"containers": ["opus", "ogg"], "audioCodecs": ["opus"], "protocols": ["http"]},
			{"containers": ["wav"], "audioCodecs": ["pcm"], "protocols": ["http"]},
			{"containers": ["dsf"], "audioCodecs": ["dsd"], "protocols": ["http"]}
		],
		"transcodingProfiles": [
			{"container": "mp3", "audioCodec": "mp3", "protocol": "http"}
		]
	}`

	// bitrateCapClient has maxAudioBitrate set to 320000 bps (320 kbps)
	bitrateCapClient = `{
		"name": "test-bitrate-cap",
		"maxAudioBitrate": 320000,
		"directPlayProfiles": [
			{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["http"]},
			{"containers": ["flac"], "audioCodecs": ["flac"], "protocols": ["http"]}
		],
		"transcodingProfiles": [
			{"container": "mp3", "audioCodec": "mp3", "protocol": "http"}
		]
	}`

	// opusTranscodeClient can direct-play mp3, transcode to opus
	opusTranscodeClient = `{
		"name": "test-opus-transcode",
		"directPlayProfiles": [
			{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["http"]}
		],
		"transcodingProfiles": [
			{"container": "opus", "audioCodec": "opus", "protocol": "http"}
		]
	}`

	// flacOnlyClient can direct-play flac, transcode to flac (no mp3 support at all)
	flacOnlyClient = `{
		"name": "test-flac-only",
		"directPlayProfiles": [
			{"containers": ["flac"], "audioCodecs": ["flac"], "protocols": ["http"]}
		],
		"transcodingProfiles": [
			{"container": "flac", "audioCodec": "flac", "protocol": "http"}
		]
	}`

	// maxTranscodeBitrateClient has maxTranscodingAudioBitrate set
	maxTranscodeBitrateClient = `{
		"name": "test-max-transcode-bitrate",
		"maxTranscodingAudioBitrate": 192000,
		"directPlayProfiles": [
			{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["http"]}
		],
		"transcodingProfiles": [
			{"container": "mp3", "audioCodec": "mp3", "protocol": "http"}
		]
	}`

	// dsdToFlacClient can direct-play mp3, transcode to flac
	dsdToFlacClient = `{
		"name": "test-dsd-to-flac",
		"directPlayProfiles": [
			{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["http"]}
		],
		"transcodingProfiles": [
			{"container": "flac", "audioCodec": "flac", "protocol": "http"}
		]
	}`
)

var _ = Describe("Transcode Endpoints", Ordered, func() {
	// Track IDs resolved in BeforeAll
	var (
		mp3TrackID       string // Come Together (mp3, 320kbps)
		flacTrackID      string // TC FLAC Standard (flac, 900kbps)
		flacHiResTrackID string // TC FLAC HiRes (flac, 3000kbps)
		alacTrackID      string // TC ALAC Track (m4a, alac)
		dsdTrackID       string // TC DSD Track (dsf, dsd)
		opusTrackID      string // TC Opus Track (opus, 128kbps)
		mkaOpusTrackID   string // TC MKA Opus (mka, opus via codec tag)
	)

	BeforeAll(func() {
		setupTestDB()

		songs, err := ds.MediaFile(ctx).GetAll()
		Expect(err).ToNot(HaveOccurred())
		byTitle := map[string]string{}
		for _, s := range songs {
			byTitle[s.Title] = s.ID
		}
		ensureGetTrackID := func(title string) string {
			id := byTitle[title]
			Expect(id).ToNot(BeEmpty())
			return id
		}
		mp3TrackID = ensureGetTrackID("Come Together")
		flacTrackID = ensureGetTrackID("TC FLAC Standard")
		flacHiResTrackID = ensureGetTrackID("TC FLAC HiRes")
		alacTrackID = ensureGetTrackID("TC ALAC Track")
		dsdTrackID = ensureGetTrackID("TC DSD Track")
		opusTrackID = ensureGetTrackID("TC Opus Track")
		mkaOpusTrackID = ensureGetTrackID("TC MKA Opus")
	})

	Describe("getTranscodeDecision", func() {
		Describe("error cases", func() {
			It("returns 405 for GET request", func() {
				w := doRawReq("getTranscodeDecision", "mediaId", mp3TrackID, "mediaType", "song")
				Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
			})

			It("returns error when mediaId is missing", func() {
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
				Expect(resp.Error.Code).To(Equal(responses.ErrorMissingParameter))
			})

			It("returns error when mediaType is missing", func() {
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", mp3TrackID)
				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
				Expect(resp.Error.Code).To(Equal(responses.ErrorMissingParameter))
			})

			It("returns error for unsupported mediaType", func() {
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", mp3TrackID, "mediaType", "video")
				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
				Expect(resp.Error.Code).To(Equal(responses.ErrorGeneric))
			})

			It("returns error for invalid JSON body", func() {
				resp := doPostReq("getTranscodeDecision", "{invalid-json", "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})

			It("returns error for empty JSON body", func() {
				w := doRawPostReq("getTranscodeDecision", "", "mediaId", mp3TrackID, "mediaType", "song")
				Expect(w.Code).To(Equal(http.StatusOK)) // Subsonic errors are returned as 200 with error status
				resp := parseJSONResponse(w)
				Expect(resp.Status).To(Equal(responses.StatusFailed))
			})

			It("returns error for non-existent media ID", func() {
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", "non-existent-id", "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
				Expect(resp.Error.Code).To(Equal(responses.ErrorDataNotFound))
			})

			It("returns error for invalid protocol in body", func() {
				invalidBody := `{
					"directPlayProfiles": [
						{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["invalid-protocol"]}
					]
				}`
				resp := doPostReq("getTranscodeDecision", invalidBody, "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})

			It("returns error for invalid comparison operator in body", func() {
				invalidBody := `{
					"directPlayProfiles": [
						{"containers": ["mp3"], "audioCodecs": ["mp3"], "protocols": ["http"]}
					],
					"codecProfiles": [{
						"type": "AudioCodec", "name": "mp3",
						"limitations": [{"name": "audioBitrate", "comparison": "InvalidOp", "values": ["320000"]}]
					}]
				}`
				resp := doPostReq("getTranscodeDecision", invalidBody, "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusFailed))
				Expect(resp.Error).ToNot(BeNil())
			})
		})

		Describe("direct play decisions", func() {
			It("allows MP3 direct play when client supports mp3", func() {
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeTrue())
				Expect(resp.TranscodeDecision.TranscodeStream).To(BeNil())
				Expect(resp.TranscodeDecision.TranscodeParams).ToNot(BeEmpty())
			})

			It("allows FLAC direct play when client supports flac", func() {
				resp := doPostReq("getTranscodeDecision", flacAndMp3Client, "mediaId", flacTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeTrue())
			})

			It("allows ALAC direct play via m4a container + alac codec matching", func() {
				resp := doPostReq("getTranscodeDecision", universalClient, "mediaId", alacTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeTrue())
			})

			It("allows Opus direct play when client supports opus", func() {
				resp := doPostReq("getTranscodeDecision", universalClient, "mediaId", opusTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeTrue())
			})

			It("denies direct play when container mismatches", func() {
				// mp3OnlyClient cannot play FLAC container
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", flacTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeFalse())
			})

			It("denies direct play when codec mismatches", func() {
				// MKA container with opus codec — client only supports mp3
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", mkaOpusTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeFalse())
			})

			It("denies direct play when maxAudioBitrate exceeded", func() {
				// bitrateCapClient caps at 320kbps, FLAC is 900kbps
				resp := doPostReq("getTranscodeDecision", bitrateCapClient, "mediaId", flacTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeFalse())
			})
		})

		Describe("transcode decisions", func() {
			It("transcodes FLAC to MP3 when client only supports MP3", func() {
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", flacTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeFalse())
				Expect(resp.TranscodeDecision.CanTranscode).To(BeTrue())
				Expect(resp.TranscodeDecision.TranscodeStream).ToNot(BeNil())
				Expect(resp.TranscodeDecision.TranscodeStream.Container).To(Equal("mp3"))
				Expect(resp.TranscodeDecision.TranscodeStream.Codec).To(Equal("mp3"))
				Expect(resp.TranscodeDecision.TranscodeParams).ToNot(BeEmpty())
			})

			It("transcodes FLAC hi-res to Opus with correct sample rate", func() {
				resp := doPostReq("getTranscodeDecision", opusTranscodeClient, "mediaId", flacHiResTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanTranscode).To(BeTrue())
				Expect(resp.TranscodeDecision.TranscodeStream).ToNot(BeNil())
				Expect(resp.TranscodeDecision.TranscodeStream.Codec).To(Equal("opus"))
				// Opus always outputs 48000 Hz
				Expect(resp.TranscodeDecision.TranscodeStream.AudioSamplerate).To(Equal(int32(48000)))
			})

			It("transcodes DSD to FLAC with normalized sample rate and bit depth", func() {
				resp := doPostReq("getTranscodeDecision", dsdToFlacClient, "mediaId", dsdTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanTranscode).To(BeTrue())
				Expect(resp.TranscodeDecision.TranscodeStream).ToNot(BeNil())
				Expect(resp.TranscodeDecision.TranscodeStream.Codec).To(Equal("flac"))
				// DSD sample rate normalized: 2822400 / 8 = 352800
				Expect(resp.TranscodeDecision.TranscodeStream.AudioSamplerate).To(Equal(int32(352800)))
				// DSD 1-bit → 24-bit PCM
				Expect(resp.TranscodeDecision.TranscodeStream.AudioBitdepth).To(Equal(int32(24)))
			})

			It("refuses lossy to lossless transcoding: MP3 to FLAC", func() {
				// flacOnlyClient can't direct-play mp3, and lossy→lossless transcode is rejected
				resp := doPostReq("getTranscodeDecision", flacOnlyClient, "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				// MP3 is lossy, FLAC is lossless — should not allow transcoding
				Expect(resp.TranscodeDecision.CanTranscode).To(BeFalse())
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeFalse())
				Expect(resp.TranscodeDecision.TranscodeParams).To(BeEmpty())
			})

			It("caps transcode bitrate via maxTranscodingAudioBitrate", func() {
				resp := doPostReq("getTranscodeDecision", maxTranscodeBitrateClient, "mediaId", flacTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				Expect(resp.TranscodeDecision.CanTranscode).To(BeTrue())
				Expect(resp.TranscodeDecision.TranscodeStream).ToNot(BeNil())
				// maxTranscodingAudioBitrate is 192000 bps = 192 kbps → response in bps
				Expect(resp.TranscodeDecision.TranscodeStream.AudioBitrate).To(Equal(int32(192000)))
			})
		})

		Describe("response structure", func() {
			It("has correct sourceStream details", func() {
				resp := doPostReq("getTranscodeDecision", universalClient, "mediaId", flacTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				src := resp.TranscodeDecision.SourceStream
				Expect(src).ToNot(BeNil())
				Expect(src.Container).To(Equal("flac"))
				Expect(src.Codec).To(Equal("flac"))
				// AudioBitrate is in bps: 900 kbps * 1000 = 900000 bps
				Expect(src.AudioBitrate).To(Equal(int32(900000)))
				Expect(src.AudioSamplerate).To(Equal(int32(44100)))
				Expect(src.AudioChannels).To(Equal(int32(2)))
				Expect(src.Protocol).To(Equal("http"))
			})

			It("reports audioBitrate in bps (kbps * 1000)", func() {
				resp := doPostReq("getTranscodeDecision", universalClient, "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				src := resp.TranscodeDecision.SourceStream
				Expect(src).ToNot(BeNil())
				// MP3 is 320 kbps → 320000 bps
				Expect(src.AudioBitrate).To(Equal(int32(320000)))
			})
		})
	})

	Describe("getTranscodeStream", func() {
		Describe("error cases", func() {
			It("returns 400 when mediaId is missing", func() {
				w := doRawReq("getTranscodeStream", "mediaType", "song", "transcodeParams", "some-token")
				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("returns 400 when mediaType is missing", func() {
				w := doRawReq("getTranscodeStream", "mediaId", mp3TrackID, "transcodeParams", "some-token")
				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("returns 400 when transcodeParams is missing", func() {
				w := doRawReq("getTranscodeStream", "mediaId", mp3TrackID, "mediaType", "song")
				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("returns 400 for unsupported mediaType", func() {
				w := doRawReq("getTranscodeStream", "mediaId", mp3TrackID, "mediaType", "video", "transcodeParams", "some-token")
				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("returns 410 for malformed token", func() {
				w := doRawReq("getTranscodeStream", "mediaId", mp3TrackID, "mediaType", "song", "transcodeParams", "invalid-token")
				Expect(w.Code).To(Equal(http.StatusGone))
			})

			It("returns 410 for stale token (media file updated after token issued)", func() {
				// Get a valid decision token
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision).ToNot(BeNil())
				token := resp.TranscodeDecision.TranscodeParams
				Expect(token).ToNot(BeEmpty())

				// Save original UpdatedAt and restore after test
				mf, err := ds.MediaFile(ctx).Get(mp3TrackID)
				Expect(err).ToNot(HaveOccurred())
				originalUpdatedAt := mf.UpdatedAt

				// Update the media file's UpdatedAt to simulate a change after token issuance
				mf.UpdatedAt = time.Now().Add(time.Hour)
				Expect(ds.MediaFile(ctx).Put(mf)).To(Succeed())

				// Attempt to stream with the now-stale token
				w := doRawReq("getTranscodeStream", "mediaId", mp3TrackID, "mediaType", "song", "transcodeParams", token)
				Expect(w.Code).To(Equal(http.StatusGone))

				// Restore original UpdatedAt
				mf.UpdatedAt = originalUpdatedAt
				Expect(ds.MediaFile(ctx).Put(mf)).To(Succeed())
			})
		})

		Describe("round-trip: decision then stream", func() {
			It("streams direct play for MP3", func() {
				// Get decision
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision.CanDirectPlay).To(BeTrue())
				token := resp.TranscodeDecision.TranscodeParams
				Expect(token).ToNot(BeEmpty())

				// Stream using the token
				w := doRawReq("getTranscodeStream", "mediaId", mp3TrackID, "mediaType", "song", "transcodeParams", token)
				Expect(w.Code).To(Equal(http.StatusOK))
				// Direct play: format should be "raw" or empty
				Expect(spy.LastRequest.Format).To(BeElementOf("raw", ""))
			})

			It("streams transcoded FLAC to MP3", func() {
				// Get decision
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", flacTrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				Expect(resp.TranscodeDecision.CanTranscode).To(BeTrue())
				token := resp.TranscodeDecision.TranscodeParams
				Expect(token).ToNot(BeEmpty())

				// Stream using the token
				w := doRawReq("getTranscodeStream", "mediaId", flacTrackID, "mediaType", "song", "transcodeParams", token)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(spy.LastRequest.Format).To(Equal("mp3"))
			})

			It("passes offset through to stream request", func() {
				// Get decision
				resp := doPostReq("getTranscodeDecision", mp3OnlyClient, "mediaId", mp3TrackID, "mediaType", "song")
				Expect(resp.Status).To(Equal(responses.StatusOK))
				token := resp.TranscodeDecision.TranscodeParams
				Expect(token).ToNot(BeEmpty())

				// Stream with offset
				w := doRawReq("getTranscodeStream", "mediaId", mp3TrackID, "mediaType", "song",
					"transcodeParams", token, "offset", "30")
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(spy.LastRequest.Offset).To(Equal(30))
			})
		})
	})
})
