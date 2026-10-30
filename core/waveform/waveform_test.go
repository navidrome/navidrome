package waveform

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// pcm encodes samples as mono s16le, the format DecodePCM produces.
func pcm(samples ...int16) string {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, samples)
	return buf.String()
}

// window returns a full window of samples where the loudest one is v.
func window(v int16) []int16 {
	w := make([]int16, windowSize)
	w[windowSize/2] = v
	return w
}

// passthroughCache calls the read function on every Get, like a disabled FileCache.
type passthroughCache struct {
	w     *waveform
	calls int
}

func (c *passthroughCache) Get(ctx context.Context, item cache.Item) (*cache.CachedStream, error) {
	c.calls++
	r, err := c.w.generate(ctx, item.(*waveformJob).mf)
	if err != nil {
		return nil, err
	}
	return &cache.CachedStream{Reader: r}, nil
}
func (c *passthroughCache) Available(context.Context) bool { return true }
func (c *passthroughCache) Disabled(context.Context) bool  { return false }

var _ = Describe("Waveform", func() {
	Describe("windowPeaks", func() {
		It("returns the max absolute amplitude of each window", func() {
			var samples []int16
			samples = append(samples, window(1000)...)
			samples = append(samples, window(-2000)...)
			samples = append(samples, 300, -32768) // partial last window
			peaks, err := windowPeaks(bytes.NewReader([]byte(pcm(samples...))))
			Expect(err).ToNot(HaveOccurred())
			Expect(peaks).To(Equal([]uint16{1000, 2000, 32768}))
		})

		It("ignores a trailing odd byte", func() {
			peaks, err := windowPeaks(bytes.NewReader(append([]byte(pcm(500)), 0x7f)))
			Expect(err).ToNot(HaveOccurred())
			Expect(peaks).To(Equal([]uint16{500}))
		})

		It("returns no peaks for empty input", func() {
			peaks, err := windowPeaks(bytes.NewReader(nil))
			Expect(err).ToNot(HaveOccurred())
			Expect(peaks).To(BeEmpty())
		})
	})

	Describe("scale", func() {
		It("maps 16-bit amplitudes to 0-255", func() {
			Expect(scale([]uint16{0, 16384, 32767, 32768})).To(Equal([]uint8{0, 128, 255, 255}))
		})
	})

	Describe("Downsample", func() {
		It("keeps the maximum of each group", func() {
			Expect(Downsample([]uint8{1, 9, 2, 3, 8, 4}, 3)).To(Equal([]uint8{9, 3, 8}))
		})

		It("handles uneven groups", func() {
			Expect(Downsample([]uint8{1, 2, 3, 4, 5}, 2)).To(Equal([]uint8{2, 5}))
		})

		It("never upsamples", func() {
			peaks := []uint8{1, 2, 3}
			Expect(Downsample(peaks, 10)).To(Equal(peaks))
			Expect(Downsample(peaks, 0)).To(Equal(peaks))
		})
	})

	Describe("encode/decode", func() {
		It("round-trips peaks", func() {
			peaks, err := decode(encode([]uint8{1, 2, 3}))
			Expect(err).ToNot(HaveOccurred())
			Expect(peaks).To(Equal([]uint8{1, 2, 3}))
		})

		It("rejects other format versions", func() {
			_, err := decode([]byte{formatVersion + 1, 1, 2})
			Expect(err).To(HaveOccurred())
			_, err = decode(nil)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Get", func() {
		var ff *tests.MockFFmpeg
		var fc *passthroughCache
		var w *waveform
		mf := &model.MediaFile{ID: "1", Path: "song.mp3"}

		BeforeEach(func() {
			conf.Update(func(c *conf.Options) { c.EnableWaveform = true })
			var samples []int16
			for _, v := range []int16{32767, 0, 16384, 0} {
				samples = append(samples, window(v)...)
			}
			ff = tests.NewMockFFmpeg(pcm(samples...))
			fc = &passthroughCache{}
			w = newWaveform(ff, fc)
			fc.w = w
		})

		It("returns the peaks at the requested resolution", func() {
			peaks, err := w.Get(GinkgoT().Context(), mf, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(peaks).To(Equal([]uint8{255, 128}))
			Expect(ff.IsClosed()).To(BeTrue())
		})

		It("returns all stored peaks when samples is 0", func() {
			peaks, err := w.Get(GinkgoT().Context(), mf, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(peaks).To(Equal([]uint8{255, 0, 128, 0}))
		})

		It("returns ErrDisabled when the feature is off", func() {
			conf.Update(func(c *conf.Options) { c.EnableWaveform = false })
			_, err := w.Get(GinkgoT().Context(), mf, 2)
			Expect(err).To(MatchError(ErrDisabled))
			Expect(fc.calls).To(BeZero())
		})

		It("returns ffmpeg errors", func() {
			ff.Error = errors.New("boom")
			_, err := w.Get(GinkgoT().Context(), mf, 2)
			Expect(err).To(MatchError("boom"))
		})
	})

	Describe("waveformJob.Key", func() {
		It("changes when the file is updated", func() {
			mf := &model.MediaFile{ID: "1", UpdatedAt: time.UnixMilli(1000)}
			k1 := (&waveformJob{mf: mf}).Key()
			mf.UpdatedAt = time.UnixMilli(2000)
			Expect((&waveformJob{mf: mf}).Key()).ToNot(Equal(k1))
		})
	})
})
