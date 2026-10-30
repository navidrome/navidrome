package waveform

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	// formatVersion is the first byte of every cached entry. Bump it whenever the peak
	// algorithm or the encoding changes, so stale entries are recomputed.
	formatVersion = 1

	// MaxSamples is the resolution waveforms are computed and stored at. Requests for
	// fewer samples are downsampled from it.
	MaxSamples = 2000

	// DefaultSamples is used when the client does not ask for a specific resolution.
	DefaultSamples = 500

	// sampleRate is the rate tracks are decoded at. It only has to resolve the amplitude
	// envelope, not the audio itself.
	sampleRate = 8000

	// windowSize is the number of PCM samples (10ms) folded into each intermediate peak,
	// keeping memory bounded (~720KB per hour of audio) regardless of track length.
	windowSize = sampleRate / 100
)

// ErrDisabled is returned when the waveform feature is turned off in the configuration.
var ErrDisabled = errors.New("waveform generation is disabled")

// Waveform computes the amplitude envelope of media files.
type Waveform interface {
	// Get returns the peaks of mf, downsampled to the given number of samples. Each peak is
	// the maximum absolute amplitude of its time slice on a linear 0-255 scale (255 = full scale).
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
			FileCache: cache.NewFileCache("Waveform", conf.Server().WaveformCacheSize, consts.WaveformCacheDir,
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
	if !conf.Server().EnableWaveform {
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

	windows, err := windowPeaks(pcm)
	if err != nil {
		return nil, fmt.Errorf("decoding audio: %w", err)
	}
	peaks := Downsample(scale(windows), MaxSamples)
	log.Debug(ctx, "Generated waveform", "id", mf.ID, "path", mf.Path, "samples", len(peaks), "elapsed", time.Since(start))
	return bytes.NewReader(encode(peaks)), nil
}

// windowPeaks reads mono s16le PCM and returns the maximum absolute amplitude of each window.
func windowPeaks(r io.Reader) ([]uint16, error) {
	br := bufio.NewReaderSize(r, 64*1024)
	var peaks []uint16
	var buf [2]byte
	var peak uint16
	n := 0
	for {
		if _, err := io.ReadFull(br, buf[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}
		v := int16(binary.LittleEndian.Uint16(buf[:]))
		a := uint16(v)
		if v < 0 {
			a = uint16(-int32(v))
		}
		peak = max(peak, a)
		n++
		if n == windowSize {
			peaks = append(peaks, peak)
			peak, n = 0, 0
		}
	}
	if n > 0 {
		peaks = append(peaks, peak)
	}
	return peaks, nil
}

// scale maps 16-bit amplitudes (0-32768) to 0-255.
func scale(amps []uint16) []uint8 {
	out := make([]uint8, len(amps))
	for i, a := range amps {
		out[i] = uint8(min(255, (uint32(a)*255+16384)/32768))
	}
	return out
}

// Downsample reduces peaks to the given number of samples, keeping the maximum of each group so
// short transients stay visible. It never upsamples: when samples >= len(peaks), peaks is returned as is.
func Downsample(peaks []uint8, samples int) []uint8 {
	n := len(peaks)
	if samples <= 0 || samples >= n {
		return peaks
	}
	out := make([]uint8, samples)
	for i := range samples {
		from, to := i*n/samples, (i+1)*n/samples
		var m uint8
		for _, p := range peaks[from:to] {
			m = max(m, p)
		}
		out[i] = m
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
