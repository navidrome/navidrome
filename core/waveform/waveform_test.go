package waveform

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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
	Describe("windowEnergy", func() {
		It("returns the mean square amplitude of each window", func() {
			var samples []int16
			samples = append(samples, constant(16384)...)
			samples = append(samples, constant(-8192)...)
			samples = append(samples, -32768) // partial last window
			energies, err := windowEnergy(bytes.NewReader([]byte(pcm(samples...))), maxWindows)
			Expect(err).ToNot(HaveOccurred())
			Expect(energies).To(Equal([]float32{0.25, 0.0625, 1}))
		})

		It("ignores a trailing odd byte", func() {
			energies, err := windowEnergy(bytes.NewReader(append([]byte(pcm(16384)), 0x7f)), maxWindows)
			Expect(err).ToNot(HaveOccurred())
			Expect(energies).To(Equal([]float32{0.25}))
		})

		It("merges windows in pairs to stay within the limit", func() {
			var samples []int16
			for _, v := range []int16{16384, 16384, 8192, 8192, 16384, 16384, 8192} {
				samples = append(samples, constant(v)...)
			}
			energies, err := windowEnergy(bytes.NewReader([]byte(pcm(samples...))), 4)
			Expect(err).ToNot(HaveOccurred())
			// The first 4 windows merge into 2 when the limit is reached; the next 2 windows fill
			// one double-size window, and the last one is a partial double-size window.
			Expect(energies).To(Equal([]float32{0.25, 0.0625, 0.25, 0.0625}))
		})

		It("never keeps more than limit values, however long the track", func() {
			silence := bytes.NewReader(make([]byte, 1000*windowSize*2))
			energies, err := windowEnergy(silence, 8)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(energies)).To(BeNumerically(">=", 4))
			Expect(cap(energies)).To(Equal(8))
		})

		It("returns no energies for empty input", func() {
			energies, err := windowEnergy(bytes.NewReader(nil), maxWindows)
			Expect(err).ToNot(HaveOccurred())
			Expect(energies).To(BeEmpty())
		})
	})

	Describe("envelope", func() {
		It("returns the RMS of each slice, relative to the loudest one", func() {
			Expect(envelope([]float32{0.25, 0.25, 0.0625, 0.0625}, 2)).To(Equal([]uint8{255, 128}))
		})

		It("averages energy, not amplitude, within a slice", func() {
			// RMS of the slice is sqrt((1+0)/2) = 0.707, relative to the loudest slice (1)
			Expect(envelope([]float32{1, 0, 1, 1}, 2)).To(Equal([]uint8{180, 255}))
		})

		It("never returns more slices than windows", func() {
			Expect(envelope([]float32{1, 0.25}, 10)).To(Equal([]uint8{255, 128}))
		})

		It("returns zeros for silence", func() {
			Expect(envelope([]float32{0, 0}, 2)).To(Equal([]uint8{0, 0}))
		})
	})

	Describe("Downsample", func() {
		It("returns the quadratic mean of each group", func() {
			Expect(Downsample([]uint8{100, 100, 255, 0, 3, 4}, 3)).To(Equal([]uint8{100, 180, 4}))
		})

		It("handles uneven groups", func() {
			Expect(Downsample([]uint8{10, 10, 10, 10, 10}, 2)).To(Equal([]uint8{10, 10}))
		})

		It("never upsamples", func() {
			values := []uint8{1, 2, 3}
			Expect(Downsample(values, 10)).To(Equal(values))
			Expect(Downsample(values, 0)).To(Equal(values))
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
			DeferCleanup(configtest.SetupConfig())
			conf.Server.EnableWaveform = true
			var samples []int16
			for _, v := range []int16{16384, 0, 8192, 0} {
				samples = append(samples, constant(v)...)
			}
			ff = tests.NewMockFFmpeg(pcm(samples...))
			fc = &passthroughCache{}
			w = newWaveform(ff, fc)
			fc.w = w
		})

		It("returns the peaks at the requested resolution", func() {
			peaks, err := w.Get(GinkgoT().Context(), mf, 2)
			Expect(err).ToNot(HaveOccurred())
			// Stored as {255, 0, 128, 0}, then each pair is combined by quadratic mean
			Expect(peaks).To(Equal([]uint8{180, 91}))
			Expect(ff.IsClosed()).To(BeTrue())
		})

		It("returns all stored peaks when samples is 0", func() {
			peaks, err := w.Get(GinkgoT().Context(), mf, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(peaks).To(Equal([]uint8{255, 0, 128, 0}))
		})

		It("returns ErrDisabled when the feature is off", func() {
			conf.Server.EnableWaveform = false
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
