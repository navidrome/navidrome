package waveform

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/singleton"
)

const (
	// formatVersion is the first byte of every cached entry. Bump it whenever the
	// algorithm or the encoding changes, so stale entries are recomputed.
	formatVersion = 2

	// MaxSamples is the resolution waveforms are computed and stored at. Requests for
	// fewer samples are downsampled from it.
	MaxSamples = 2000

	// DefaultSamples is used when the client does not ask for a specific resolution.
	DefaultSamples = 500

	// sampleRate is the rate tracks are decoded at. It only has to resolve the amplitude
	// envelope, not the audio itself.
	sampleRate = 8000

	// windowSize is the number of PCM samples (10ms) folded into each intermediate energy value.
	windowSize = sampleRate / 100

	// maxWindows bounds the energy values kept while decoding. When it is reached, neighbouring
	// windows are merged in pairs and later windows cover twice as many samples, so memory stays
	// around 64KB whatever the track length, with at least 4x MaxSamples windows left to envelope.
	maxWindows = 8 * MaxSamples
)

// ErrDisabled is returned when the waveform feature is turned off in the configuration.
var ErrDisabled = errors.New("waveform generation is disabled")

// Waveform computes the amplitude envelope of media files.
type Waveform interface {
	// Get returns the loudness envelope of mf, downsampled to the given number of samples. Each
	// value is the RMS amplitude of its time slice on a linear 0-255 scale, relative to the
	// loudest slice of the track (255). RMS is used instead of the peak amplitude because mastered
	// music reaches full scale almost everywhere, which makes a peak envelope flat.
	Get(ctx context.Context, mf *model.MediaFile, samples int) ([]uint8, error)
}

type waveform struct {
	ffmpeg ffmpeg.FFmpeg
	cache  cache.FileCache
	slots  chan struct{}
}

// New returns the Waveform service. The cache is shared process-wide, like the image and
// transcoding caches.
func New(ff ffmpeg.FFmpeg) Waveform {
	return newWaveform(ff, GetWaveformCache())
}

func newWaveform(ff ffmpeg.FFmpeg, fc cache.FileCache) *waveform {
	return &waveform{
		ffmpeg: ff,
		cache:  fc,
		slots:  make(chan struct{}, max(1, runtime.NumCPU()/2)),
	}
}

func GetWaveformCache() cache.FileCache {
	return singleton.GetInstance(func() *waveformCache {
		return &waveformCache{
			FileCache: cache.NewFileCache("Waveform", conf.Server.WaveformCacheSize, consts.WaveformCacheDir,
				consts.DefaultWaveformCacheMaxItems,
				func(ctx context.Context, arg cache.Item) (io.Reader, error) {
					job := arg.(*waveformJob)
					return job.w.generate(ctx, job.mf)
				}),
		}
	})
}

type waveformCache struct {
	cache.FileCache
}

type waveformJob struct {
	w  *waveform
	mf *model.MediaFile
}

// Key includes the file's modification time, so a replaced or re-tagged file gets a fresh waveform.
func (j *waveformJob) Key() string {
	return fmt.Sprintf("%s.%d.v%d", j.mf.ID, j.mf.UpdatedAt.UnixMilli(), formatVersion)
}

func (w *waveform) Get(ctx context.Context, mf *model.MediaFile, samples int) ([]uint8, error) {
	if !conf.Server.EnableWaveform {
		return nil, ErrDisabled
	}
	s, err := w.cache.Get(ctx, &waveformJob{w: w, mf: mf})
	if err != nil {
		return nil, err
	}
	defer s.Close()
	data, err := io.ReadAll(s)
	if err != nil {
		return nil, err
	}
	peaks, err := decode(data)
	if err != nil {
		return nil, err
	}
	return Downsample(peaks, samples), nil
}

func (w *waveform) generate(ctx context.Context, mf *model.MediaFile) (io.Reader, error) {
	select {
	case w.slots <- struct{}{}:
		defer func() { <-w.slots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	start := time.Now()
	pcm, err := w.ffmpeg.DecodePCM(ctx, mf.AbsolutePath(), sampleRate)
	if err != nil {
		return nil, err
	}
	defer pcm.Close()

	windows, err := windowEnergy(pcm, maxWindows)
	if err != nil {
		return nil, fmt.Errorf("decoding audio: %w", err)
	}
	peaks := envelope(windows, MaxSamples)
	log.Debug(ctx, "Generated waveform", "id", mf.ID, "path", mf.Path, "samples", len(peaks), "elapsed", time.Since(start))
	return bytes.NewReader(encode(peaks)), nil
}

// windowEnergy reads mono s16le PCM and returns the mean square amplitude (0-1) of each window.
// Windows start at windowSize samples. Whenever limit (an even number) of them are stored, they
// are merged in pairs and the window size doubles, so at most limit values are ever kept.
func windowEnergy(r io.Reader, limit int) ([]float32, error) {
	br := bufio.NewReaderSize(r, 64*1024)
	energies := make([]float32, 0, limit)
	size := windowSize
	var buf [2]byte
	var sum float64
	n := 0
	for {
		if _, err := io.ReadFull(br, buf[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}
		v := float64(int16(binary.LittleEndian.Uint16(buf[:]))) / 32768
		sum += v * v
		n++
		if n == size {
			energies = append(energies, float32(sum/float64(n)))
			sum, n = 0, 0
			if len(energies) == limit {
				energies = mergePairs(energies)
				size *= 2
			}
		}
	}
	if n > 0 {
		energies = append(energies, float32(sum/float64(n)))
	}
	return energies, nil
}

// mergePairs averages neighbouring values in place and returns the first half of energies.
// Both values of a pair cover the same number of samples, so the average stays their mean square.
func mergePairs(energies []float32) []float32 {
	half := len(energies) / 2
	for i := range half {
		energies[i] = (energies[2*i] + energies[2*i+1]) / 2
	}
	return energies[:half]
}

// envelope groups window energies into at most samples slices and returns the RMS amplitude of
// each, scaled so the loudest slice is 255. A silent track is all zeros.
func envelope(energies []float32, samples int) []uint8 {
	n := len(energies)
	samples = min(samples, n)
	rms := make([]float64, samples)
	var loudest float64
	for i := range samples {
		from, to := i*n/samples, (i+1)*n/samples
		var sum float64
		for _, e := range energies[from:to] {
			sum += float64(e)
		}
		rms[i] = math.Sqrt(sum / float64(to-from))
		loudest = max(loudest, rms[i])
	}
	out := make([]uint8, samples)
	if loudest == 0 {
		return out
	}
	for i, v := range rms {
		out[i] = uint8(math.Round(v / loudest * 255))
	}
	return out
}

// Downsample reduces values to the given number of samples. Each output is the quadratic mean of
// the values it covers, so it stays the RMS of the longer slice. It never upsamples: when
// samples >= len(values), values is returned as is.
func Downsample(values []uint8, samples int) []uint8 {
	n := len(values)
	if samples <= 0 || samples >= n {
		return values
	}
	out := make([]uint8, samples)
	for i := range samples {
		from, to := i*n/samples, (i+1)*n/samples
		var sum float64
		for _, v := range values[from:to] {
			sum += float64(v) * float64(v)
		}
		out[i] = uint8(math.Round(math.Sqrt(sum / float64(to-from))))
	}
	return out
}

func encode(peaks []uint8) []byte {
	return append([]byte{formatVersion}, peaks...)
}

func decode(data []byte) ([]uint8, error) {
	if len(data) == 0 || data[0] != formatVersion {
		return nil, errors.New("invalid cached waveform")
	}
	return data[1:], nil
}
