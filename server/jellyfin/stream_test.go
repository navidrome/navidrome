package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stream", func() {
	var api *Router
	var ds *tests.MockDataStore
	var streamer *fakeMediaStreamer
	var decider *fakeTranscodeDecider

	// alice has access to library 1 only.
	ctxUser := func() context.Context {
		return request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice", Libraries: model.Libraries{{ID: 1, Name: "Music"}}})
	}

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		streamer = &fakeMediaStreamer{}
		decider = &fakeTranscodeDecider{}
		api = &Router{ds: ds, streamer: streamer, transcodeDecider: decider}
	})

	Describe("getPlaybackInfo", func() {
		It("returns a media source for an accessible track", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", Duration: 100, Size: 1000, LibraryID: 1},
			})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Items/s1/PlaybackInfo", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			api.getPlaybackInfo(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.PlaybackInfoResponse
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.MediaSources).To(HaveLen(1))
			Expect(res.MediaSources[0].Id).To(Equal("s1"))
			Expect(res.MediaSources[0].Container).To(Equal("mp3"))
			Expect(res.PlaySessionId).ToNot(BeEmpty())
		})

		It("returns 404 for a track in a library the user can't access", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 2},
			})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Items/s1/PlaybackInfo", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", "s1")
			api.getPlaybackInfo(w, r)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 404 when the id doesn't match any media file", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Items/missing/PlaybackInfo", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "missing")
			api.getPlaybackInfo(w, r)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("streamAudio", func() {
		It("invokes the transcode decider and streamer for an accessible track", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 1},
			})
			streamer.content = "audio-bytes"
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/s1/stream", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			api.streamAudio(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(decider.invoked).To(BeTrue())
			Expect(streamer.invoked).To(BeTrue())
			Expect(w.Body.String()).To(Equal("audio-bytes"))
		})

		It("returns 404 for a track in a library the user can't access, without invoking the streamer or decider", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 2},
			})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/s1/stream", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", "s1")
			api.streamAudio(w, r)

			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(decider.invoked).To(BeFalse())
			Expect(streamer.invoked).To(BeFalse())
		})

		It("returns 404 when the id doesn't match any media file, without invoking the streamer or decider", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/missing/stream", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "missing")
			api.streamAudio(w, r)

			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(decider.invoked).To(BeFalse())
			Expect(streamer.invoked).To(BeFalse())
		})

		It("returns 500 and logs when the streamer fails", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 1},
			})
			streamer.err = errors.New("boom")
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/s1/stream", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			api.streamAudio(w, r)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("streamFile", func() {
		It("invokes the decider with a raw/direct-play request and the streamer for an accessible track", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 1},
			})
			streamer.content = "audio-bytes"
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/s1/File", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			api.streamFile(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(decider.invoked).To(BeTrue())
			Expect(decider.req.Format).To(Equal("raw"))
			Expect(streamer.invoked).To(BeTrue())
			Expect(w.Body.String()).To(Equal("audio-bytes"))
		})

		It("returns 404 for a track in a library the user can't access, without invoking the streamer or decider", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 2},
			})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/s1/File", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", "s1")
			api.streamFile(w, r)

			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(decider.invoked).To(BeFalse())
			Expect(streamer.invoked).To(BeFalse())
		})

		It("returns 404 when the id doesn't match any media file, without invoking the streamer or decider", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/missing/File", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "missing")
			api.streamFile(w, r)

			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(decider.invoked).To(BeFalse())
			Expect(streamer.invoked).To(BeFalse())
		})
	})
})

// fakeTranscodeDecider is a local test double for stream.TranscodeDecider: it records whether
// (and how) ResolveRequest was invoked, so tests can assert it's never called on the
// access-denied path, without needing a real transcode decision pipeline.
type fakeTranscodeDecider struct {
	invoked bool
	req     stream.Request
}

func (f *fakeTranscodeDecider) MakeDecision(context.Context, *model.MediaFile, *stream.ClientInfo, stream.TranscodeOptions) (*stream.TranscodeDecision, error) {
	return &stream.TranscodeDecision{}, nil
}

func (f *fakeTranscodeDecider) CreateTranscodeParams(*stream.TranscodeDecision) (string, error) {
	return "", nil
}

func (f *fakeTranscodeDecider) ResolveRequestFromToken(context.Context, string, *model.MediaFile, int) (stream.Request, error) {
	return stream.Request{}, nil
}

func (f *fakeTranscodeDecider) ResolveRequest(_ context.Context, _ *model.MediaFile, format string, bitRate int, offset int) stream.Request {
	f.invoked = true
	f.req = stream.Request{Format: format, BitRate: bitRate, Offset: offset}
	return f.req
}

// fakeMediaStreamer is a local test double for stream.MediaStreamer: it records whether
// NewStream was invoked and, on success, returns a real (non-seekable) *stream.Stream backed
// by an in-memory reader, so streamAudio's call to Stream.Serve exercises real code.
type fakeMediaStreamer struct {
	invoked bool
	content string
	err     error
}

func (f *fakeMediaStreamer) NewStream(_ context.Context, mf *model.MediaFile, _ stream.Request) (*stream.Stream, error) {
	f.invoked = true
	if f.err != nil {
		return nil, f.err
	}
	return stream.NewStream(mf, mf.Suffix, 0, io.NopCloser(strings.NewReader(f.content))), nil
}
