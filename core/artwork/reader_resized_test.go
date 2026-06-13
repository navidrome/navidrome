package artwork

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("resizeImage", func() {
	var mockFF *tests.MockFFmpeg
	var r *resizedArtworkReader

	BeforeEach(func() {
		mockFF = tests.NewMockFFmpeg("converted-animated-data")
		r = &resizedArtworkReader{
			size:   300,
			square: false,
			a:      &artwork{ffmpeg: mockFF},
		}
	})

	Describe("animated GIF handling", func() {
		It("converts animated GIF via ffmpeg when available", func() {
			data := createAnimatedGIF(3)
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			// Should have been processed by ffmpeg (mock returns "converted-animated-data")
			output, err := io.ReadAll(result)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(data)) // MockFFmpeg echoes input back
		})

		It("falls back to static resize when ffmpeg fails for animated GIF", func() {
			mockFF.Error = errors.New("ffmpeg failed")
			// Use size smaller than image so static resize actually produces output
			r.size = 1
			data := createAnimatedGIF(3)
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			// Should fall through to static resize successfully (no ffmpeg error propagated)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			// Verify it's a static image (WebP encoded), not the ffmpeg error
			output, err := io.ReadAll(result)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(output)).To(BeNumerically(">", 0))
		})

		It("preserves animation for square thumbnails with animated GIF", func() {
			r.square = true
			data := createAnimatedGIF(3)
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			// Should have been processed by ffmpeg (mock returns input data)
			output, err := io.ReadAll(result)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(data))
		})
	})

	Describe("animated WebP handling", func() {
		It("returns animated WebP data as-is when not square", func() {
			data := createAnimatedWebPBytes()
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			// Should return original data unchanged
			output, err := io.ReadAll(result)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(data))
		})

		It("preserves animated WebP for square thumbnails", func() {
			r.square = true
			data := createAnimatedWebPBytes()
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			// Should return original data unchanged
			output, err := io.ReadAll(result)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(data))
		})
	})

	Describe("animated PNG handling", func() {
		It("returns animated PNG data as-is when not square", func() {
			data := createAPNGBytes()
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			// Should return original data unchanged
			output, err := io.ReadAll(result)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(data))
		})

		It("preserves animated PNG for square thumbnails", func() {
			r.square = true
			data := createAPNGBytes()
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			// Should return original data unchanged
			output, err := io.ReadAll(result)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(data))
		})
	})

	Describe("static image handling", func() {
		It("resizes a static PNG normally", func() {
			data := createStaticPNGBytes()
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			// Static PNG is 2x2, size 300 is larger, so should return nil (no upscale)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("ReadCloser preservation", func() {
		It("preserves Close semantics from ffmpeg ReadCloser", func() {
			// Create a trackable ReadCloser
			tracker := &closeTracker{Reader: bytes.NewReader([]byte("test data"))}
			mockFF2 := &mockFFmpegWithCloser{tracker: tracker}
			r.a = &artwork{ffmpeg: mockFF2}

			data := createAnimatedGIF(3)
			result, _, err := r.resizeImage(context.Background(), bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())

			// The result should be an io.ReadCloser (the tracker)
			rc, ok := result.(io.ReadCloser)
			Expect(ok).To(BeTrue())
			Expect(rc.Close()).ToNot(HaveOccurred())
			Expect(tracker.closed).To(BeTrue())
		})
	})
})

// closeTracker is an io.ReadCloser that tracks whether Close was called.
type closeTracker struct {
	io.Reader
	closed bool
}

func (c *closeTracker) Close() error {
	c.closed = true
	return nil
}

// mockFFmpegWithCloser is a minimal FFmpeg mock that returns a specific ReadCloser
// for ConvertAnimatedImage, allowing us to verify Close propagation.
type mockFFmpegWithCloser struct {
	ffmpeg.FFmpeg
	tracker *closeTracker
}

func (m *mockFFmpegWithCloser) IsAvailable() bool { return true }
func (m *mockFFmpegWithCloser) ConvertAnimatedImage(_ context.Context, _ io.Reader, _ int, _ int) (io.ReadCloser, error) {
	return m.tracker, nil
}
