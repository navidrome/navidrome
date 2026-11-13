package stream_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing/iotest"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/navidrome/navidrome/conf"
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
		cacheDir, _ := os.MkdirTemp("", "file_caches")
		conf.Update(func(c *conf.Options) {
			c.CacheFolder = conf.NewDir(cacheDir)
			c.TranscodingCacheSize = "100MB"
		})
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		ds.MediaFile().(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "123", Path: "tests/fixtures/test.mp3", Suffix: "mp3", BitRate: 128, Duration: 257.0},
		})
		testCache := stream.NewTranscodingCache()
		Eventually(func() bool { return testCache.Available(context.TODO()) }, 10*time.Second).Should(BeTrue())
		streamer = stream.NewMediaStreamer(ds, ffmpeg, testCache)
	})
	AfterEach(func() {
		_ = os.RemoveAll(conf.Server().CacheFolder.String())
	})

	Context("NewStream", func() {
		var mf *model.MediaFile
		BeforeEach(func() {
			var err error
			mf, err = ds.MediaFile().Get(ctx, "123")
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

			conf.Update(func(c *conf.Options) {
				c.Transcoding.MaxConcurrent = 1
				c.Transcoding.MaxConcurrentPerUser = 0
			})
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
			conf.Update(func(c *conf.Options) {
				c.Transcoding.MaxConcurrent = 1
				c.Transcoding.MaxConcurrentPerUser = 0
			})
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
			conf.Update(func(c *conf.Options) {
				c.Transcoding.MaxConcurrent = 1
				c.Transcoding.MaxConcurrentPerUser = 0
			})
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
			mf, err = ds.MediaFile().Get(ctx, "123")
			Expect(err).ToNot(HaveOccurred())
		})

		It("keeps empty output a non-error, so callers still reply 200 with an empty body", func() {
			s := stream.NewStream(mf, "mp3", 128, io.NopCloser(bytes.NewReader(nil)))
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			n, err := s.Serve(ctx, w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(BeZero())
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("aborts the response when the source fails after sending data", func() {
			src := io.NopCloser(io.MultiReader(
				bytes.NewReader(bytes.Repeat([]byte("a"), 64*1024)),
				iotest.ErrReader(errors.New("transcoder died")),
			))
			server := httptest.NewServer(serveHandler(stream.NewStream(mf, "mp3", 128, src)))
			DeferCleanup(server.Close)

			resp, err := http.Get(server.URL)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			// A client-side read failure is the only observable proof the response was aborted.
			_, err = io.ReadAll(resp.Body)
			Expect(err).To(HaveOccurred())
		})

		Context("with estimateContentLength=true", func() {
			// 10s at 128kbps, where a kbps is 1000 bits per second.
			const estimate = 10 * 128 * 1000 / 8

			BeforeEach(func() {
				mf.Duration = 10
			})

			newStream := func(src []byte) *stream.Stream {
				return stream.NewStream(mf, "mp3", 128, io.NopCloser(bytes.NewReader(src)))
			}
			serve := func(s *stream.Stream) *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/?estimateContentLength=true", nil)
				_, err := s.Serve(ctx, w, r)
				Expect(err).ToNot(HaveOccurred())
				return w
			}
			declaredLength := func(w *httptest.ResponseRecorder) int {
				declared, err := strconv.Atoi(w.Header().Get("Content-Length"))
				Expect(err).ToNot(HaveOccurred(), "Content-Length must be declared")
				return declared
			}

			It("pads output shorter than the estimate with zeros up to the declared length", func() {
				src := bytes.Repeat([]byte("a"), 1000)
				w := serve(newStream(src))

				declared, body := declaredLength(w), w.Body.Bytes()
				Expect(body).To(HaveLen(declared))
				Expect(body[:len(src)]).To(Equal(src))
				Expect(body[len(src):]).To(Equal(make([]byte, declared-len(src))))
			})

			It("truncates output longer than the estimate at the declared length", func() {
				src := bytes.Repeat([]byte("a"), 2*estimate)
				w := serve(newStream(src))

				declared := declaredLength(w)
				Expect(declared).To(BeNumerically("<", len(src)))
				Expect(w.Body.Bytes()).To(Equal(src[:declared]))
			})

			It("reads the rest of output longer than the estimate, so a transcode tied to the request can finish", func() {
				src := bytes.NewReader(bytes.Repeat([]byte("a"), 2*estimate))
				serve(stream.NewStream(mf, "mp3", 128, io.NopCloser(src)))

				Expect(src.Len()).To(BeZero())
			})

			It("declares duration times bitrate, with no headroom", func() {
				w := serve(newStream(bytes.Repeat([]byte("a"), 1000)))

				Expect(declaredLength(w)).To(Equal(estimate))
			})

			It("estimates only the part of the track after the requested offset", func() {
				full, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 128})
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(full.Close)
				half, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 128, Offset: 5})
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(half.Close)

				Expect(half.EstimatedContentLength()).To(BeNumerically("~", full.EstimatedContentLength()/2, 1))
			})

			It("estimates the whole track for a negative offset, as the transcode starts from the beginning", func() {
				s, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 128, Offset: -1000000000})
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(s.Close)

				Expect(s.EstimatedContentLength()).To(Equal(int64(estimate)))
			})

			It("does not declare a Content-Length when the track has no duration", func() {
				mf.Duration = 0
				src := []byte("some audio")
				w := serve(newStream(src))

				Expect(w.Header().Values("Content-Length")).To(BeEmpty())
				Expect(w.Body.Bytes()).To(Equal(src))
			})

			It("drops the declared length when the transcoder produces no output", func() {
				w := serve(newStream(nil))

				Expect(w.Header().Values("Content-Length")).To(BeEmpty())
				Expect(w.Body.Len()).To(BeZero())
			})

			Context("with DevMatchEstimatedContentLength=false", func() {
				BeforeEach(func() {
					conf.Update(func(c *conf.Options) { c.DevMatchEstimatedContentLength = false })
				})

				It("declares the estimate and sends shorter output as is, without padding", func() {
					src := bytes.Repeat([]byte("a"), 1000)
					w := serve(newStream(src))

					Expect(declaredLength(w)).To(Equal(estimate))
					Expect(w.Body.Bytes()).To(Equal(src))
				})

				It("does not cut output longer than the estimate", func() {
					src := bytes.Repeat([]byte("a"), 2*estimate)
					w := serve(newStream(src))

					Expect(w.Body.Bytes()).To(Equal(src))
				})
			})
		})
	})
})

// Serve runs behind the real server's Recoverer, which must let ErrAbortHandler through.
func serveHandler(s *stream.Stream) http.Handler {
	return middleware.Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = s.Serve(r.Context(), w, r)
	}))
}
