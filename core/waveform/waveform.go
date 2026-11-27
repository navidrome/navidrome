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
	"slices"
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
	formatVersion = 3

	// IntervalMs is the length of audio, in milliseconds, covered by each stored level.
	IntervalMs = 100

	// Silence is the lowest level, in hundredths of dBFS (-120 dBFS). Quieter windows,
	// including digital silence, are clamped to it.
	Silence = -12000

	// sampleRate is the rate tracks are decoded at. It only has to resolve the loudness
	// envelope, not the audio itself.
	sampleRate = 8000

	// windowSize is the number of PCM samples in each IntervalMs window.
	windowSize = sampleRate * IntervalMs / 1000
)

// ErrDisabled is returned when the waveform feature is turned off in the configuration.
var ErrDisabled = errors.New("waveform generation is disabled")

// Waveform computes the loudness envelope of media files.
type Waveform interface {
	// Get returns the RMS level of each IntervalMs window of mf, in playback order, in hundredths
	// of dBFS, from Silence to 0. RMS is used instead of the peak amplitude because mastered music
	// reaches full scale almost everywhere, which makes a peak envelope flat.
	Get(ctx context.Context, mf *model.MediaFile) ([]int16, error)
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

func (w *waveform) Get(ctx context.Context, mf *model.MediaFile) ([]int16, error) {
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
	return decode(data)
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

	levels, err := windowLevels(pcm, int(mf.Duration*1000/IntervalMs)+1)
	if err != nil {
		return nil, fmt.Errorf("decoding audio: %w", err)
	}
	log.Debug(ctx, "Generated waveform", "id", mf.ID, "path", mf.Path, "levels", len(levels), "elapsed", time.Since(start))
	return bytes.NewReader(encode(levels)), nil
}

// windowLevels reads mono s16le PCM and returns the RMS level of each windowSize window, in
// hundredths of dBFS. The last window can be shorter. sizeHint preallocates the result.
func windowLevels(r io.Reader, sizeHint int) ([]int16, error) {
	br := bufio.NewReaderSize(r, 64*1024)
	levels := make([]int16, 0, max(sizeHint, 0))
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
		if n == windowSize {
			levels = append(levels, level(sum/float64(n)))
			sum, n = 0, 0
		}
	}
	if n > 0 {
		levels = append(levels, level(sum/float64(n)))
	}
	return levels, nil
}

// level converts a mean square amplitude (0-1) to hundredths of dBFS, clamped to [Silence, 0].
func level(meanSquare float64) int16 {
	if meanSquare <= 0 {
		return Silence
	}
	db := math.Round(10 * math.Log10(meanSquare) * 100)
	return int16(min(max(db, Silence), 0))
}

// Downsample reduces levels to exactly samples values. Value i is the highest of levels
// [i*n/samples, (i+1)*n/samples), so short loud parts stay visible. It never upsamples: when
// samples <= 0 or samples >= len(levels), levels is returned as is.
func Downsample(levels []int16, samples int) []int16 {
	n := len(levels)
	if samples <= 0 || samples >= n {
		return levels
	}
	out := make([]int16, samples)
	for i := range samples {
		from, to := i*n/samples, (i+1)*n/samples
		out[i] = slices.Max(levels[from:to])
	}
	return out
}

// encode stores levels as the format version followed by little-endian int16 values.
func encode(levels []int16) []byte {
	out := make([]byte, 1, 1+2*len(levels))
	out[0] = formatVersion
	for _, v := range levels {
		out = binary.LittleEndian.AppendUint16(out, uint16(v))
	}
	return out
}

func decode(data []byte) ([]int16, error) {
	if len(data) == 0 || data[0] != formatVersion || len(data)%2 != 1 {
		return nil, errors.New("invalid cached waveform")
	}
	levels := make([]int16, (len(data)-1)/2)
	for i := range levels {
		levels[i] = int16(binary.LittleEndian.Uint16(data[1+2*i:]))
	}
	return levels, nil
}
