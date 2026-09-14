package artwork

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"sync/atomic"
	"testing/iotest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("resizeStaticImage", func() {
	It("rejects images whose declared dimensions exceed the pixel cap before decoding", func() {
		_, _, err := resizeStaticImage(pngHeaderWithDims(9000, 9000), 300, false)
		Expect(err).To(MatchError(ContainSubstring("exceed pixel cap")))
	})
})

var _ = Describe("resizeImageData", func() {
	var gifBytes []byte

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableWebPEncoding = false
		gifBytes = createAnimatedGIF(3)
	})

	It("converts an animated GIF with ffmpeg", func() {
		fake := &animFFmpeg{MockFFmpeg: tests.NewMockFFmpeg(""), out: []byte("animated-webp")}
		r, _, err := resizeImageData(context.Background(), fake, gifBytes, 1, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(io.ReadAll(r)).To(Equal([]byte("animated-webp")))
	})

	DescribeTable("falls back to a static resize when ffmpeg fails",
		func(fake *animFFmpeg) {
			fake.MockFFmpeg = tests.NewMockFFmpeg("")
			r, _, err := resizeImageData(context.Background(), fake, gifBytes, 1, false)
			Expect(err).ToNot(HaveOccurred())
			data, err := io.ReadAll(r)
			Expect(err).ToNot(HaveOccurred())
			_, format, err := image.DecodeConfig(bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())
			Expect(format).To(Equal("jpeg"))
		},
		Entry("before producing output", &animFFmpeg{err: errors.New("no libwebp_anim")}),
		Entry("mid-stream, like ffmpeg 5.1 reading a GIF from a pipe", &animFFmpeg{streamErr: errors.New("pipe:0: Input/output error")}),
		Entry("with empty output", &animFFmpeg{}),
	)
})

// animFFmpeg fakes animated conversion; a non-nil release blocks it until closed.
type animFFmpeg struct {
	*tests.MockFFmpeg
	calls     atomic.Int32
	release   chan struct{}
	out       []byte
	err       error
	streamErr error
}

func (f *animFFmpeg) ConvertAnimatedImage(ctx context.Context, _ io.Reader, _ int, _ int) (io.ReadCloser, error) {
	f.calls.Add(1)
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	if f.streamErr != nil {
		return io.NopCloser(iotest.ErrReader(f.streamErr)), nil //nolint:nilerr // the stream fails, not the call
	}
	return io.NopCloser(bytes.NewReader(f.out)), nil
}
