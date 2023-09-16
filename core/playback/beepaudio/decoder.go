//go:build beep

package beepaudio

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

func DecodeMp3(path string) (s beep.StreamSeekCloser, format beep.Format, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	return mp3.Decode(f)
}

func DecodeWAV(path string) (s beep.StreamSeekCloser, format beep.Format, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	return wav.Decode(f)
}

func DecodeFLAC(path string) (s beep.StreamSeekCloser, format beep.Format, fileToCleanup string, err error) {
	// TODO: Turn this into a semi-parallel operation: start playing while still transcoding/copying
	log.Debug("decode to FLAC", "filename", path)
	fFmpeg := ffmpeg.New()
	readCloser, err := fFmpeg.ConvertToFLAC(context.TODO(), path)
	if err != nil {
		log.Error("error converting file to FLAC", path, err)
		return nil, beep.Format{}, "", err
	}

	tempFile, err := os.CreateTemp("", "*.flac")

	if err != nil {
		log.Error("error creating temp file", err)
		return nil, beep.Format{}, "", err
	}
	log.Debug("created tempfile", "filename", tempFile.Name())

	written, err := io.Copy(tempFile, readCloser)
	if err != nil {
		log.Error("error coping file", "dest", tempFile.Name())
	}
	log.Debug("copy pipe into tempfile", "bytes written", written, "filename", tempFile.Name())

	f, err := os.Open(tempFile.Name())
	if err != nil {
		log.Error("could not re-open tempfile", "filename", tempFile.Name())
		return nil, beep.Format{}, "", err
	}

	s, format, err = flac.Decode(f)
	return s, format, tempFile.Name(), err
}
