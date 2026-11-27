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

// constant returns a full window of samples all set to v.
func constant(v int16) []int16 {
	w := make([]int16, windowSize)
	for i := range w {
		w[i] = v
	}
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
	Describe("windowLevels", func() {
		It("returns the RMS level of each window in hundredths of dBFS", func() {
			var samples []int16
			samples = append(samples, constant(16384)...)
			samples = append(samples, constant(-8192)...)
			samples = append(samples, -32768) // partial last window
			levels, err := windowLevels(bytes.NewReader([]byte(pcm(samples...))), 0)
			Expect(err).ToNot(HaveOccurred())
			// 0.5 and 0.25 of full scale are -6.02 and -12.04 dBFS; a full-scale sample is 0 dBFS
			Expect(levels).To(Equal([]int16{-602, -1204, 0}))
		})

		It("covers 100 ms per window", func() {
			Expect(windowSize).To(Equal(sampleRate / 10))
		})

		It("returns -120 dBFS for digital silence", func() {
			levels, err := windowLevels(bytes.NewReader([]byte(pcm(constant(0)...))), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(levels).To(Equal([]int16{Silence}))
		})
	})

	Describe("level", func() {
		It("clamps levels below -120 dBFS", func() {
			Expect(level(1e-13)).To(Equal(int16(Silence)))
		})

		It("rounds to hundredths of a dB", func() {
			Expect(level(0.5)).To(Equal(int16(-301)))
			Expect(level(1)).To(Equal(int16(0)))
		})
		It("ignores a trailing odd byte", func() {
			levels, err := windowLevels(bytes.NewReader(append([]byte(pcm(16384)), 0x7f)), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(levels).To(Equal([]int16{-602}))
		})

		It("returns no levels for empty input", func() {
			levels, err := windowLevels(bytes.NewReader(nil), 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(levels).To(BeEmpty())
		})
	})

	Describe("Downsample", func() {
		It("returns the highest level of each group", func() {
			Expect(Downsample([]int16{-1000, -900, -12000, -300, -3000, -4000}, 3)).To(Equal([]int16{-900, -300, -3000}))
		})

		It("handles uneven groups", func() {
			// 5 values into 2: groups are [0, 2) and [2, 5)
			Expect(Downsample([]int16{-100, -200, -300, -400, -500}, 2)).To(Equal([]int16{-100, -300}))
		})

		It("never upsamples", func() {
			levels := []int16{-1, -2, -3}
			Expect(Downsample(levels, 3)).To(Equal(levels))
			Expect(Downsample(levels, 10)).To(Equal(levels))
			Expect(Downsample(levels, 0)).To(Equal(levels))
		})
	})

	Describe("encode/decode", func() {
		It("round-trips levels", func() {
			levels, err := decode(encode([]int16{Silence, -3142, 0}))
			Expect(err).ToNot(HaveOccurred())
			Expect(levels).To(Equal([]int16{Silence, -3142, 0}))
		})

		It("rejects other format versions and truncated data", func() {
			_, err := decode([]byte{formatVersion + 1, 1, 2})
			Expect(err).To(HaveOccurred())
			_, err = decode([]byte{formatVersion, 1})
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
			for _, v := range []int16{16384, 0, 8192, 0} {
				samples = append(samples, constant(v)...)
			}
			ff = tests.NewMockFFmpeg(pcm(samples...))
			fc = &passthroughCache{}
			w = newWaveform(ff, fc)
			fc.w = w
		})

		It("returns the level of every 100 ms window", func() {
			levels, err := w.Get(GinkgoT().Context(), mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(levels).To(Equal([]int16{-602, Silence, -1204, Silence}))
			Expect(ff.IsClosed()).To(BeTrue())
		})

		It("returns ErrDisabled when the feature is off", func() {
			conf.Update(func(c *conf.Options) { c.EnableWaveform = false })
			_, err := w.Get(GinkgoT().Context(), mf)
			Expect(err).To(MatchError(ErrDisabled))
			Expect(fc.calls).To(BeZero())
		})

		It("returns ffmpeg errors", func() {
			ff.Error = errors.New("boom")
			_, err := w.Get(GinkgoT().Context(), mf)
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
