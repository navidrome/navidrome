package stream_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

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
		Eventually(func() bool { return testCache.Available(context.TODO()) }, 10*time.Second).Should(BeTrue())
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
			Eventually(func() bool { return tightCache.Available(context.TODO()) }, 10*time.Second).Should(BeTrue())
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
			Eventually(func() bool { return tightCache.Available(context.TODO()) }, 10*time.Second).Should(BeTrue())
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
			Eventually(func() bool { return tightCache.Available(context.TODO()) }, 10*time.Second).Should(BeTrue())
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
	})

	Context("Serve", func() {
		var mf *model.MediaFile
		BeforeEach(func() {
			var err error
			mf, err = ds.MediaFile(ctx).Get("123")
			Expect(err).ToNot(HaveOccurred())
		})

		It("aborts the response when the source fails after sending data", func() {
			src := &failingReader{data: bytes.Repeat([]byte("a"), 64*1024), err: errors.New("no data available")}
			server := httptest.NewServer(serveHandler(stream.NewStream(mf, "mp3", 128, src)))
			DeferCleanup(server.Close)

			resp, err := http.Get(server.URL)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			_, err = io.ReadAll(resp.Body)
			Expect(err).To(HaveOccurred())
		})

		It("keeps empty output a non-error, so callers still reply 200 with an empty body", func() {
			src := &truncatedSource{Reader: bytes.NewReader(nil), err: errors.New("no such codec")}
			s := stream.NewStream(mf, "mp3", 128, src)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			n, err := s.Serve(ctx, w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(BeZero())
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("aborts the response when the source reports a failure after a clean EOF", func() {
			src := &truncatedSource{
				Reader: bytes.NewReader(bytes.Repeat([]byte("a"), 64*1024)),
				err:    errors.New("transcoder died"),
			}
			server := httptest.NewServer(serveHandler(stream.NewStream(mf, "mp3", 128, src)))
			DeferCleanup(server.Close)

			resp, err := http.Get(server.URL)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			_, err = io.ReadAll(resp.Body)
			Expect(err).To(HaveOccurred())
		})
	})
})

// serveHandler exercises Serve through the same Recoverer used by the real server,
// so an aborted response is only observable if the middleware lets the panic through.
func serveHandler(s *stream.Stream) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = s.Serve(r.Context(), w, r)
	})
	return r
}

type failingReader struct {
	data []byte
	err  error
	pos  int
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, r.err
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *failingReader) Close() error { return nil }

// truncatedSource reads to a clean EOF but reports that the data behind it is incomplete.
type truncatedSource struct {
	io.Reader
	err error
}

func (r *truncatedSource) Close() error { return nil }
func (r *truncatedSource) Err() error   { return r.err }
