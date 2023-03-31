package playback

import (
	"bytes"
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

func decodeFLAC(ctx context.Context, path string) (s beep.StreamSeekCloser, format beep.Format, err error) {
	fFmpeg := ffmpeg.New()
	readCloser, err := fFmpeg.ConvertToFLAC(ctx, path)
	if err != nil {
		log.Error(err)
		return
	}

	data, err := io.ReadAll(readCloser)
	if err != nil {
		log.Error(err)
		return
	}
	return flac.Decode(bytes.NewReader(data))
}
