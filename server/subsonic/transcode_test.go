package subsonic

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/transcode"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transcode endpoints", func() {
	var (
		router     *Router
		ds         *tests.MockDataStore
		mockTD     *mockTranscodeDecision
		w          *httptest.ResponseRecorder
		mockMFRepo *tests.MockMediaFileRepo
	)

	BeforeEach(func() {
		mockMFRepo = &tests.MockMediaFileRepo{}
		ds = &tests.MockDataStore{MockedMediaFile: mockMFRepo}
		mockTD = &mockTranscodeDecision{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockTD)
		w = httptest.NewRecorder()
	})

	Describe("GetTranscodeDecision", func() {
		It("returns 405 for non-POST requests", func() {
			r := newGetRequest("mediaId=123", "mediaType=song")
			resp, err := router.GetTranscodeDecision(w, r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(BeNil())
			Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
			Expect(w.Header().Get("Allow")).To(Equal("POST"))
		})

		It("returns error when mediaId is missing", func() {
			r := newJSONPostRequest("mediaType=song", "{}")
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when mediaType is missing", func() {
			r := newJSONPostRequest("mediaId=123", "{}")
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for unsupported mediaType", func() {
			r := newJSONPostRequest("mediaId=123&mediaType=podcast", "{}")
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not yet supported"))
		})

		It("returns error when media file not found", func() {
			mockMFRepo.SetError(true)
			r := newJSONPostRequest("mediaId=notfound&mediaType=song", "{}")
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when body is empty", func() {
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", "")
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when body contains invalid JSON", func() {
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", "not-json{{{")
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid protocol in direct play profile", func() {
			body := `{"directPlayProfiles":[{"containers":["mp3"],"audioCodecs":["mp3"],"protocols":["ftp"]}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid protocol"))
		})

		It("returns error for invalid comparison operator", func() {
			body := `{"codecProfiles":[{"type":"AudioCodec","name":"mp3","limitations":[{"name":"audioBitrate","comparison":"InvalidOp","values":["320"]}]}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid comparison"))
		})

		It("returns error for invalid limitation name", func() {
			body := `{"codecProfiles":[{"type":"AudioCodec","name":"mp3","limitations":[{"name":"unknownField","comparison":"Equals","values":["320"]}]}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid limitation name"))
		})

		It("returns error for invalid codec profile type", func() {
			body := `{"codecProfiles":[{"type":"VideoCodec","name":"mp3"}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid codec profile type"))
		})

		It("rejects wrong-case protocol", func() {
			body := `{"directPlayProfiles":[{"containers":["mp3"],"audioCodecs":["mp3"],"protocols":["HTTP"]}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid protocol"))
		})

		It("rejects wrong-case codec profile type", func() {
			body := `{"codecProfiles":[{"type":"audiocodec","name":"mp3"}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid codec profile type"))
		})

		It("rejects wrong-case comparison operator", func() {
			body := `{"codecProfiles":[{"type":"AudioCodec","name":"mp3","limitations":[{"name":"audioBitrate","comparison":"lessthanequal","values":["320"]}]}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid comparison"))
		})

		It("rejects wrong-case limitation name", func() {
			body := `{"codecProfiles":[{"type":"AudioCodec","name":"mp3","limitations":[{"name":"AudioBitrate","comparison":"Equals","values":["320"]}]}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			_, err := router.GetTranscodeDecision(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid limitation name"))
		})

		It("returns a valid decision response", func() {
			mockMFRepo.SetData(model.MediaFiles{
				{ID: "song-1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100},
			})
			mockTD.decision = &transcode.Decision{
				MediaID:       "song-1",
				CanDirectPlay: true,
				SourceStream: transcode.StreamDetails{
					Container: "mp3", Codec: "mp3", Bitrate: 320,
					SampleRate: 44100, Channels: 2,
				},
			}
			mockTD.token = "test-jwt-token"

			body := `{"directPlayProfiles":[{"containers":["mp3"],"protocols":["http"]}]}`
			r := newJSONPostRequest("mediaId=song-1&mediaType=song", body)
			resp, err := router.GetTranscodeDecision(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.TranscodeDecision).ToNot(BeNil())
			Expect(resp.TranscodeDecision.CanDirectPlay).To(BeTrue())
			Expect(resp.TranscodeDecision.TranscodeParams).To(Equal("test-jwt-token"))
			Expect(resp.TranscodeDecision.SourceStream).ToNot(BeNil())
			Expect(resp.TranscodeDecision.SourceStream.Protocol).To(Equal("http"))
			Expect(resp.TranscodeDecision.SourceStream.Container).To(Equal("mp3"))
			Expect(resp.TranscodeDecision.SourceStream.AudioBitrate).To(Equal(int32(320_000)))
		})

		It("includes transcode stream when transcoding", func() {
			mockMFRepo.SetData(model.MediaFiles{
				{ID: "song-2", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24},
			})
			mockTD.decision = &transcode.Decision{
				MediaID:          "song-2",
				CanDirectPlay:    false,
				CanTranscode:     true,
				TargetFormat:     "mp3",
				TargetBitrate:    256,
				TranscodeReasons: []string{"container not supported"},
				SourceStream: transcode.StreamDetails{
					Container: "flac", Codec: "flac", Bitrate: 1000,
					SampleRate: 96000, BitDepth: 24, Channels: 2,
				},
				TranscodeStream: &transcode.StreamDetails{
					Container: "mp3", Codec: "mp3", Bitrate: 256,
					SampleRate: 96000, Channels: 2,
				},
			}
			mockTD.token = "transcode-token"

			r := newJSONPostRequest("mediaId=song-2&mediaType=song", "{}")
			resp, err := router.GetTranscodeDecision(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.TranscodeDecision.CanTranscode).To(BeTrue())
			Expect(resp.TranscodeDecision.TranscodeReasons).To(ConsistOf("container not supported"))
			Expect(resp.TranscodeDecision.TranscodeStream).ToNot(BeNil())
			Expect(resp.TranscodeDecision.TranscodeStream.Container).To(Equal("mp3"))
		})
	})

	Describe("GetTranscodeStream", func() {
		It("returns error when mediaId is missing", func() {
			r := newGetRequest("mediaType=song", "transcodeParams=abc")
			_, err := router.GetTranscodeStream(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when transcodeParams is missing", func() {
			r := newGetRequest("mediaId=123", "mediaType=song")
			_, err := router.GetTranscodeStream(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid token", func() {
			mockTD.parseErr = model.ErrNotFound
			r := newGetRequest("mediaId=123", "mediaType=song", "transcodeParams=bad-token")
			_, err := router.GetTranscodeStream(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when mediaId doesn't match token", func() {
			mockTD.params = &transcode.Params{MediaID: "other-id", DirectPlay: true}
			r := newGetRequest("mediaId=wrong-id", "mediaType=song", "transcodeParams=valid-token")
			_, err := router.GetTranscodeStream(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not match"))
		})

		It("builds correct StreamRequest for direct play", func() {
			fakeStreamer := &fakeMediaStreamer{}
			router = New(ds, nil, fakeStreamer, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockTD)
			mockTD.params = &transcode.Params{MediaID: "song-1", DirectPlay: true}

			r := newGetRequest("mediaId=song-1", "mediaType=song", "transcodeParams=valid-token")
			_, _ = router.GetTranscodeStream(w, r)

			Expect(fakeStreamer.captured).ToNot(BeNil())
			Expect(fakeStreamer.captured.ID).To(Equal("song-1"))
			Expect(fakeStreamer.captured.Format).To(BeEmpty())
			Expect(fakeStreamer.captured.BitRate).To(BeZero())
			Expect(fakeStreamer.captured.SampleRate).To(BeZero())
			Expect(fakeStreamer.captured.BitDepth).To(BeZero())
			Expect(fakeStreamer.captured.Channels).To(BeZero())
		})

		It("builds correct StreamRequest for transcoding", func() {
			fakeStreamer := &fakeMediaStreamer{}
			router = New(ds, nil, fakeStreamer, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mockTD)
			mockTD.params = &transcode.Params{
				MediaID:          "song-2",
				DirectPlay:       false,
				TargetFormat:     "mp3",
				TargetBitrate:    256,
				TargetSampleRate: 44100,
				TargetBitDepth:   16,
				TargetChannels:   2,
			}

			r := newGetRequest("mediaId=song-2", "mediaType=song", "transcodeParams=valid-token", "offset=10")
			_, _ = router.GetTranscodeStream(w, r)

			Expect(fakeStreamer.captured).ToNot(BeNil())
			Expect(fakeStreamer.captured.ID).To(Equal("song-2"))
			Expect(fakeStreamer.captured.Format).To(Equal("mp3"))
			Expect(fakeStreamer.captured.BitRate).To(Equal(256))
			Expect(fakeStreamer.captured.SampleRate).To(Equal(44100))
			Expect(fakeStreamer.captured.BitDepth).To(Equal(16))
			Expect(fakeStreamer.captured.Channels).To(Equal(2))
			Expect(fakeStreamer.captured.Offset).To(Equal(10))
		})
	})

	Describe("bpsToKbps", func() {
		It("converts standard bitrates", func() {
			Expect(bpsToKbps(128000)).To(Equal(128))
			Expect(bpsToKbps(320000)).To(Equal(320))
			Expect(bpsToKbps(256000)).To(Equal(256))
		})
		It("returns 0 for 0", func() {
			Expect(bpsToKbps(0)).To(Equal(0))
		})
		It("rounds instead of truncating", func() {
			Expect(bpsToKbps(999)).To(Equal(1))
			Expect(bpsToKbps(500)).To(Equal(1))
			Expect(bpsToKbps(499)).To(Equal(0))
		})
	})

	Describe("kbpsToBps", func() {
		It("converts standard bitrates", func() {
			Expect(kbpsToBps(128)).To(Equal(128000))
			Expect(kbpsToBps(320)).To(Equal(320000))
		})
		It("returns 0 for 0", func() {
			Expect(kbpsToBps(0)).To(Equal(0))
		})
	})

	Describe("convertBitrateValues", func() {
		It("converts valid bps strings to kbps", func() {
			Expect(convertBitrateValues([]string{"128000", "320000"})).To(Equal([]string{"128", "320"}))
		})
		It("preserves unparseable values", func() {
			Expect(convertBitrateValues([]string{"128000", "bad", "320000"})).To(Equal([]string{"128", "bad", "320"}))
		})
		It("handles empty slice", func() {
			Expect(convertBitrateValues([]string{})).To(Equal([]string{}))
		})
	})
})

// newJSONPostRequest creates an HTTP POST request with JSON body and query params
func newJSONPostRequest(queryParams string, jsonBody string) *http.Request {
	r := httptest.NewRequest("POST", "/getTranscodeDecision?"+queryParams, bytes.NewBufferString(jsonBody))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// mockTranscodeDecision is a test double for core.TranscodeDecision
type mockTranscodeDecision struct {
	decision *transcode.Decision
	token    string
	tokenErr error
	params   *transcode.Params
	parseErr error
}

func (m *mockTranscodeDecision) MakeDecision(_ context.Context, _ *model.MediaFile, _ *transcode.ClientInfo) (*transcode.Decision, error) {
	if m.decision != nil {
		return m.decision, nil
	}
	return &transcode.Decision{}, nil
}

func (m *mockTranscodeDecision) CreateTranscodeParams(_ *transcode.Decision) (string, error) {
	return m.token, m.tokenErr
}

func (m *mockTranscodeDecision) ParseTranscodeParams(_ string) (*transcode.Params, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.params, nil
}

// fakeMediaStreamer captures the StreamRequest and returns a sentinel error,
// allowing tests to verify parameter passing without constructing a real Stream.
var errStreamCaptured = errors.New("stream request captured")

type fakeMediaStreamer struct {
	captured *core.StreamRequest
}

func (f *fakeMediaStreamer) NewStream(_ context.Context, req core.StreamRequest) (*core.Stream, error) {
	f.captured = &req
	return nil, errStreamCaptured
}

func (f *fakeMediaStreamer) DoStream(_ context.Context, _ *model.MediaFile, req core.StreamRequest) (*core.Stream, error) {
	f.captured = &req
	return nil, errStreamCaptured
}
