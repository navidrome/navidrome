package playback

import (
	"context"
	"io"
	"os"

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

func decodeFLAC(ctx context.Context, path string) (s beep.StreamSeekCloser, format beep.Format, err error, fileToCleanup string) {
	fFmpeg := ffmpeg.New()
	readCloser, err := fFmpeg.ConvertToFLAC(ctx, path)
	if err != nil {
		log.Error(ctx, "error converting file to FLAC", path, err)
		return nil, beep.Format{}, err, ""
	}

	tempFile, err := os.CreateTemp("", "*.flac")

	if err != nil {
		log.Error(ctx, "error creating temp file", err)
		return nil, beep.Format{}, err, ""
	}
	log.Debug(ctx, "created tempfile", "filename", tempFile.Name())

	written, err := io.Copy(tempFile, readCloser)
	if err != nil {
		return nil, beep.Format{}, err, ""
	}
	log.Debug(ctx, "written # bytes to file: ", written)

	f, err := os.Open(tempFile.Name())
	if err != nil {
		return nil, beep.Format{}, err, ""
	}

	s, format, err = flac.Decode(f)
	return s, format, err, tempFile.Name()
}
