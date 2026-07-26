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
		api = &Router{
			ds: ds, streamer: streamer, transcodeDecider: decider,
			lyrics:      &fakeLyricsService{lyrics: map[string]model.LyricList{}},
			lyricsCache: newTestLyricsCache(),
		}
	})

	Describe("getPlaybackInfo", func() {
		It("returns a media source for an accessible track", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", Duration: 100, Size: 1000, LibraryID: 1},
			})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("s1")+"/PlaybackInfo", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID("s1"))
			api.getPlaybackInfo(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.PlaybackInfoResponse
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.MediaSources).To(HaveLen(1))
			Expect(res.MediaSources[0].Id).To(Equal(dto.EncodeID("s1")))
			Expect(res.MediaSources[0].Container).To(Equal("mp3"))
			Expect(res.MediaSources[0].Size).To(Equal(int64(1000)))
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

		playbackInfo := func() dto.PlaybackInfoResponse {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("s1")+"/PlaybackInfo", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID("s1"))
			api.getPlaybackInfo(w, r)
			var res dto.PlaybackInfoResponse
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			return res
		}

		lyricStreams := func(res dto.PlaybackInfoResponse) []dto.MediaStream {
			var out []dto.MediaStream
			for _, s := range res.MediaSources[0].MediaStreams {
				if s.Type == "Lyric" {
					out = append(out, s)
				}
			}
			return out
		}

		It("advertises a Lyric stream for plugin/sidecar-sourced lyrics not embedded in the file", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 1},
			})
			api.lyrics = &fakeLyricsService{lyrics: map[string]model.LyricList{
				"s1": {{Kind: "main", Synced: true, Line: []model.Line{{Value: "hello"}}}},
			}}

			Expect(lyricStreams(playbackInfo())).To(HaveLen(1))
		})

		It("advertises no Lyric stream when the pipeline finds nothing", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 1},
			})

			Expect(lyricStreams(playbackInfo())).To(BeEmpty())
		})

		It("advertises no Lyric stream when the lyrics endpoint would 404 (main lyric has no lines)", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 1},
			})
			api.lyrics = &fakeLyricsService{lyrics: map[string]model.LyricList{
				"s1": {{Kind: "main", Lang: "eng"}},
			}}

			Expect(lyricStreams(playbackInfo())).To(BeEmpty())
		})

		It("doesn't duplicate the Lyric stream when lyrics are already embedded", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 1, Lyrics: `[{"lang":"xxx","line":[]}]`},
			})
			api.lyrics = &fakeLyricsService{lyrics: map[string]model.LyricList{
				"s1": {{Kind: "main", Synced: true, Line: []model.Line{{Value: "hello"}}}},
			}}

			Expect(lyricStreams(playbackInfo())).To(HaveLen(1))
		})

		It("still returns 200 with a valid MediaSource and no Lyric stream when the lyrics pipeline errors", func() {
			// Own ID: an erroring loader isn't cached, but a shared ID could still pick up
			// another test's cached (non-error) result and mask this assertion.
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s-err", Title: "Song", Suffix: "mp3", Duration: 100, Size: 1000, LibraryID: 1},
			})
			api.lyrics = &fakeLyricsService{err: errors.New("boom")}

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("s-err")+"/PlaybackInfo", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", dto.EncodeID("s-err"))
			api.getPlaybackInfo(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.PlaybackInfoResponse
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.MediaSources).To(HaveLen(1))
			Expect(res.MediaSources[0].Id).To(Equal(dto.EncodeID("s-err")))
			Expect(lyricStreams(res)).To(BeEmpty())
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
			invoke(api.streamAudio, w, r)

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
			invoke(api.streamAudio, w, r)

			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(decider.invoked).To(BeFalse())
			Expect(streamer.invoked).To(BeFalse())
		})

		It("returns 404 when the id doesn't match any media file, without invoking the streamer or decider", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/missing/stream", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "missing")
			invoke(api.streamAudio, w, r)

			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(decider.invoked).To(BeFalse())
			Expect(streamer.invoked).To(BeFalse())
		})

		It("converts the bps audioBitRate param to kbps", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "flac", LibraryID: 1},
			})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/s1/stream?audiobitrate=320000", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.streamAudio, w, r)

			Expect(decider.req.BitRate).To(Equal(320))
		})

		It("uses the audioCodec param as target format when no container is given", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "flac", LibraryID: 1},
			})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/s1/stream?audiocodec=aac", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.streamAudio, w, r)

			Expect(decider.req.Format).To(Equal("aac"))
		})

		It("returns 500 and logs when the streamer fails", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "mp3", LibraryID: 1},
			})
			streamer.err = errors.New("boom")
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/s1/stream", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.streamAudio, w, r)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("streamHls", func() {
		BeforeEach(func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "dsf", Duration: 100.5, LibraryID: 1},
			})
		})

		hls := func(query string, ctx context.Context) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Audio/s1/main.m3u8"+query, nil).WithContext(ctx)
			r = withChiURLParam(r, "itemId", "s1")
			invoke(api.streamHls, w, r)
			return w
		}

		It("returns a single-segment VOD playlist pointing at the progressive stream endpoint", func() {
			w := hls("?audiocodec=aac&audiobitrate=320000&api_key=tok", ctxUser())

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/vnd.apple.mpegurl"))
			body := w.Body.String()
			Expect(body).To(HavePrefix("#EXTM3U\n"))
			Expect(body).To(ContainSubstring("#EXT-X-PLAYLIST-TYPE:VOD\n"))
			Expect(body).To(ContainSubstring("#EXT-X-TARGETDURATION:101\n"))
			Expect(body).To(ContainSubstring("#EXTINF:100.500,\n"))
			Expect(body).To(ContainSubstring("\nstream.aac?api_key=tok&audioBitRate=320000\n"))
			Expect(body).To(HaveSuffix("#EXT-X-ENDLIST\n"))
		})

		It("omits the bitrate param when the client doesn't send one", func() {
			w := hls("?audiocodec=aac&api_key=tok", ctxUser())
			Expect(w.Body.String()).To(ContainSubstring("\nstream.aac?api_key=tok\n"))
		})

		It("falls back to aac for codecs HLS packed-audio can't carry", func() {
			w := hls("?audiocodec=opus", ctxUser())
			Expect(w.Body.String()).To(ContainSubstring("\nstream.aac\n"))
		})

		It("honors mp3 as segment codec", func() {
			w := hls("?audiocodec=mp3", ctxUser())
			Expect(w.Body.String()).To(ContainSubstring("\nstream.mp3\n"))
		})

		It("prefers the server-forced transcoding format over the requested codec", func() {
			ctx := request.WithTranscoding(ctxUser(), model.Transcoding{TargetFormat: "mp3"})
			w := hls("?audiocodec=aac", ctx)
			Expect(w.Body.String()).To(ContainSubstring("\nstream.mp3\n"))
		})

		It("advertises an HLS-incompatible forced format verbatim, matching what the segment will contain", func() {
			ctx := request.WithTranscoding(ctxUser(), model.Transcoding{TargetFormat: "opus"})
			w := hls("?audiocodec=aac", ctx)
			Expect(w.Body.String()).To(ContainSubstring("\nstream.opus\n"))
		})

		It("returns 404 for a track in a library the user can't access", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
				{ID: "s1", Title: "Song", Suffix: "dsf", LibraryID: 2},
			})
			Expect(hls("", ctxUser()).Code).To(Equal(http.StatusNotFound))
		})

		It("returns 404 when the id doesn't match any media file", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{})
			Expect(hls("", ctxUser()).Code).To(Equal(http.StatusNotFound))
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
