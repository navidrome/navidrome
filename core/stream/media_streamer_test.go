package stream_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaStreamer", func() {
	var streamer stream.MediaStreamer
	var ds model.DataStore
	ffmpeg := tests.NewMockFFmpeg("fake data")
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		cacheDir, _ := os.MkdirTemp("", "file_caches")
		conf.Server.CacheFolder = conf.NewDir(cacheDir)
		conf.Server.TranscodingCacheSize = "100MB"
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "123", Path: "tests/fixtures/test.mp3", Suffix: "mp3", BitRate: 128, Duration: 257.0},
		})
		testCache := stream.NewTranscodingCache()
		Eventually(func() bool { return testCache.Available(context.TODO()) }).Should(BeTrue())
		streamer = stream.NewMediaStreamer(ds, ffmpeg, testCache)
	})
	AfterEach(func() {
		_ = os.RemoveAll(conf.Server.CacheFolder.String())
	})

	Context("NewStream", func() {
		var mf *model.MediaFile
		BeforeEach(func() {
			var err error
			mf, err = ds.MediaFile(ctx).Get("123")
			Expect(err).ToNot(HaveOccurred())
		})
		It("returns a seekable stream if format is 'raw'", func() {
			s, err := streamer.NewStream(ctx, mf, stream.Request{Format: "raw"})
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a seekable stream if no format is specified (direct play)", func() {
			s, err := streamer.NewStream(ctx, mf, stream.Request{})
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a NON seekable stream if transcode is required", func() {
			s, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 64})
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeFalse())
			Expect(s.Duration()).To(Equal(float32(257.0)))
		})
		It("rejects transcode requests beyond MaxConcurrent with ErrTooManyTranscodes", func() {
			// Use an ffmpeg whose Read blocks indefinitely so the cache's
			// background copy can't drain the source and release the slot —
			// keeping the single transcode slot pinned for this test.
			pr, pw := io.Pipe()
			DeferCleanup(func() { _ = pw.Close() })
			blockingFFmpeg := tests.NewMockFFmpeg("")
			blockingFFmpeg.Reader = pr

			conf.Server.Transcoding.MaxConcurrent = 1
			conf.Server.Transcoding.MaxConcurrentPerUser = 0
			tightCache := stream.NewTranscodingCache()
			Eventually(func() bool { return tightCache.Available(context.TODO()) }).Should(BeTrue())
			tightStreamer := stream.NewMediaStreamer(ds, blockingFFmpeg, tightCache)

			userCtx := request.WithUsername(ctx, "alice")
			s1, err := tightStreamer.NewStream(userCtx, mf, stream.Request{Format: "mp3", BitRate: 64})
			Expect(err).ToNot(HaveOccurred())
			defer s1.Close()

			// Different cache key so it doesn't dedupe with the first request.
			_, err = tightStreamer.NewStream(userCtx, mf, stream.Request{Format: "mp3", BitRate: 96})
			Expect(errors.Is(err, stream.ErrTooManyTranscodes)).To(BeTrue())
		})

		It("releases the slot once the stream is closed", func() {
			conf.Server.Transcoding.MaxConcurrent = 1
			conf.Server.Transcoding.MaxConcurrentPerUser = 0
			tightCache := stream.NewTranscodingCache()
			Eventually(func() bool { return tightCache.Available(context.TODO()) }).Should(BeTrue())
			tightStreamer := stream.NewMediaStreamer(ds, ffmpeg, tightCache)

			userCtx := request.WithUsername(ctx, "alice")
			s1, err := tightStreamer.NewStream(userCtx, mf, stream.Request{Format: "mp3", BitRate: 64})
			Expect(err).ToNot(HaveOccurred())
			_, _ = io.ReadAll(s1)
			_ = s1.Close()
			Eventually(func() bool { return ffmpeg.IsClosed() }, "3s").Should(BeTrue())

			// Slot should now be free for a different transcode.
			s2, err := tightStreamer.NewStream(userCtx, mf, stream.Request{Format: "mp3", BitRate: 96})
			Expect(err).ToNot(HaveOccurred())
			defer s2.Close()
		})

		It("does not consume a slot for raw streams", func() {
			conf.Server.Transcoding.MaxConcurrent = 1
			conf.Server.Transcoding.MaxConcurrentPerUser = 0
			tightCache := stream.NewTranscodingCache()
			Eventually(func() bool { return tightCache.Available(context.TODO()) }).Should(BeTrue())
			tightStreamer := stream.NewMediaStreamer(ds, ffmpeg, tightCache)

			userCtx := request.WithUsername(ctx, "alice")
			// First, saturate the single transcode slot.
			s1, err := tightStreamer.NewStream(userCtx, mf, stream.Request{Format: "mp3", BitRate: 64})
			Expect(err).ToNot(HaveOccurred())
			defer s1.Close()

			// Raw stream must still succeed.
			s2, err := tightStreamer.NewStream(userCtx, mf, stream.Request{Format: "raw"})
			Expect(err).ToNot(HaveOccurred())
			defer s2.Close()
		})

		It("returns a seekable stream if the file is complete in the cache", func() {
			s, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 32})
			Expect(err).To(BeNil())
			_, _ = io.ReadAll(s)
			_ = s.Close()
			Eventually(func() bool { return ffmpeg.IsClosed() }, "3s").Should(BeTrue())

			s, err = streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 32})
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeTrue())
		})

		It("revalidates a repeated transcoded request with 304 instead of re-sending the audio", func() {
			// Dedicated ffmpeg/cache so the transcoded payload is deterministic
			// (the suite-level mock's reader is single-use and shared).
			freshFFmpeg := tests.NewMockFFmpeg("some transcoded audio payload")
			localCache := stream.NewTranscodingCache()
			Eventually(func() bool { return localCache.Available(context.TODO()) }).Should(BeTrue())
			localStreamer := stream.NewMediaStreamer(ds, freshFFmpeg, localCache)

			// Prime the cache so the transcode is complete and seekable.
			s, err := localStreamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 32})
			Expect(err).ToNot(HaveOccurred())
			_, _ = io.ReadAll(s)
			_ = s.Close()
			Eventually(func() bool { return freshFFmpeg.IsClosed() }, "3s").Should(BeTrue())

			// First request: serve fully and capture the advertised ETag.
			s, err = localStreamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 32})
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
			rec := httptest.NewRecorder()
			_, err = s.Serve(ctx, rec, httptest.NewRequest(http.MethodGet, "/stream", nil))
			Expect(err).ToNot(HaveOccurred())
			_ = s.Close()
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.Len()).To(BeNumerically(">", 0))
			etag := rec.Header().Get("ETag")
			Expect(etag).To(HavePrefix(`W/"`))

			// Repeat with a matching If-None-Match: must be a bodiless 304.
			s, err = localStreamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 32})
			Expect(err).ToNot(HaveOccurred())
			rec2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/stream", nil)
			r2.Header.Set("If-None-Match", etag)
			_, err = s.Serve(ctx, rec2, r2)
			Expect(err).ToNot(HaveOccurred())
			_ = s.Close()
			Expect(rec2.Code).To(Equal(http.StatusNotModified))
			Expect(rec2.Body.Len()).To(Equal(0))
		})

		It("changes the ETag when the requested bitrate changes", func() {
			s1, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 32})
			Expect(err).ToNot(HaveOccurred())
			rec1 := httptest.NewRecorder()
			_, _ = s1.Serve(ctx, rec1, httptest.NewRequest(http.MethodGet, "/stream", nil))
			_ = s1.Close()

			s2, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 64})
			Expect(err).ToNot(HaveOccurred())
			rec2 := httptest.NewRecorder()
			_, _ = s2.Serve(ctx, rec2, httptest.NewRequest(http.MethodGet, "/stream", nil))
			_ = s2.Close()

			Expect(rec1.Header().Get("ETag")).ToNot(BeEmpty())
			Expect(rec2.Header().Get("ETag")).ToNot(BeEmpty())
			Expect(rec1.Header().Get("ETag")).ToNot(Equal(rec2.Header().Get("ETag")))
		})
	})
})
