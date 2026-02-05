package subsonic

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/core"
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

		It("handles empty body gracefully", func() {
			mockMFRepo.SetData(model.MediaFiles{
				{ID: "song-1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100},
			})
			mockTD.decision = &core.Decision{
				MediaID:       "song-1",
				CanDirectPlay: false,
				SourceStream: core.StreamDetails{
					Container: "mp3", Codec: "mp3", Bitrate: 320,
					SampleRate: 44100, Channels: 2,
				},
			}
			mockTD.token = "empty-body-token"

			r := newJSONPostRequest("mediaId=song-1&mediaType=song", "")
			resp, err := router.GetTranscodeDecision(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.TranscodeDecision).ToNot(BeNil())
			Expect(resp.TranscodeDecision.TranscodeParams).To(Equal("empty-body-token"))
		})

		It("returns a valid decision response", func() {
			mockMFRepo.SetData(model.MediaFiles{
				{ID: "song-1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100},
			})
			mockTD.decision = &core.Decision{
				MediaID:       "song-1",
				CanDirectPlay: true,
				SourceStream: core.StreamDetails{
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
			mockTD.decision = &core.Decision{
				MediaID:          "song-2",
				CanDirectPlay:    false,
				CanTranscode:     true,
				TargetFormat:     "mp3",
				TargetBitrate:    256,
				TranscodeReasons: []string{"container not supported"},
				SourceStream: core.StreamDetails{
					Container: "flac", Codec: "flac", Bitrate: 1000,
					SampleRate: 96000, BitDepth: 24, Channels: 2,
				},
				TranscodeStream: &core.StreamDetails{
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
			mockTD.params = &core.TranscodeParams{MediaID: "other-id", DirectPlay: true}
			r := newGetRequest("mediaId=wrong-id", "mediaType=song", "transcodeParams=valid-token")
			_, err := router.GetTranscodeStream(w, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not match"))
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
	decision *core.Decision
	token    string
	tokenErr error
	params   *core.TranscodeParams
	parseErr error
}

func (m *mockTranscodeDecision) MakeDecision(_ context.Context, _ *model.MediaFile, _ *core.ClientInfo) (*core.Decision, error) {
	if m.decision != nil {
		return m.decision, nil
	}
	return &core.Decision{}, nil
}

func (m *mockTranscodeDecision) CreateToken(_ *core.Decision) (string, error) {
	return m.token, m.tokenErr
}

func (m *mockTranscodeDecision) ParseToken(_ string) (*core.TranscodeParams, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.params, nil
}
