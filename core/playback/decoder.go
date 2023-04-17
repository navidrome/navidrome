package playback

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/wav"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
)

func decodeMp3(path string) (s beep.StreamSeekCloser, format beep.Format, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	return mp3.Decode(f)
}

func decodeWAV(path string) (s beep.StreamSeekCloser, format beep.Format, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	return wav.Decode(f)
}

func decodeFLAC(ctx context.Context, path string) (s beep.StreamSeekCloser, format beep.Format, fileToCleanup string, err error) {
	fFmpeg := ffmpeg.New()
	readCloser, err := fFmpeg.ConvertToFLAC(ctx, path)
	if err != nil {
		log.Error(ctx, "error converting file to FLAC", path, err)
		return nil, beep.Format{}, "", err
	}

	tempFile, err := os.CreateTemp("", "*.flac")

	if err != nil {
		log.Error(ctx, "error creating temp file", err)
		return nil, beep.Format{}, "", err
	}
	log.Debug(ctx, "created tempfile", "filename", tempFile.Name())

	go func() {
		written, err := io.Copy(tempFile, readCloser)
		if err != nil {
			log.Error(ctx, "error coping file", "dest", tempFile.Name())
		}
		log.Debug(ctx, "copy pipe into tempfile", "bytes written", written, "filename", tempFile.Name())
	}()

	// FIXME: turn this into a wait-for-certain-amount-of-bytes-loop
	time.Sleep(time.Second)

	f, err := os.Open(tempFile.Name())
	if err != nil {
		log.Error(ctx, "could not re-open tempfile", "filename", tempFile.Name())
		return nil, beep.Format{}, "", err
	}

	s, format, err = flac.Decode(f)
	return s, format, tempFile.Name(), err
}
