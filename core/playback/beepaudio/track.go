//go:build beep

package beepaudio

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type BeepTrack struct {
	MediaFile         model.MediaFile
	Ctrl              *beep.Ctrl
	Volume            *effects.Volume
	ActiveStream      beep.StreamSeekCloser
	TempfileToCleanup string
	SampleRate        beep.SampleRate
	PlaybackDone      chan bool
}

func NewTrack(playbackDoneChannel chan bool, mf model.MediaFile) (*BeepTrack, error) {
	t := BeepTrack{}

	contentType := mf.ContentType()
	log.Debug("loading track", "trackname", mf.Path, "mediatype", contentType)

	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error
	var tmpfileToCleanup = ""

	switch contentType {
	case "audio/mpeg":
		streamer, format, err = DecodeMp3(mf.Path)
	case "audio/x-wav":
		streamer, format, err = DecodeWAV(mf.Path)
	case "audio/mp4":
		streamer, format, tmpfileToCleanup, err = DecodeFLAC(mf.Path)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	if err != nil {
		log.Error(err)
		return nil, err
	}

	// save running stream for closing when switching tracks
	t.ActiveStream = streamer
	t.TempfileToCleanup = tmpfileToCleanup

	log.Debug("Setting up audio device")
	t.Ctrl = &beep.Ctrl{Streamer: streamer, Paused: true}
	t.Volume = &effects.Volume{Streamer: t.Ctrl, Base: 2}
	t.SampleRate = format.SampleRate
	t.PlaybackDone = playbackDoneChannel
	t.MediaFile = mf

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Error(err)
	}
	log.Debug("speaker.Init() finished")

	go func() {
		speaker.Play(beep.Seq(t.Volume, beep.Callback(func() {
			log.Info("Hitting end-of-stream, signalling on channel")
			t.PlaybackDone <- true
			log.Debug("Signalling finished")
		})))
		log.Debug("dropping out of speaker.Play()")
	}()
	return &t, nil
}

func (t *BeepTrack) String() string {
	return fmt.Sprintf("Name: %s", t.MediaFile.Path)
}

func (t *BeepTrack) SetVolume(value float64) {
	speaker.Lock()
	t.Volume.Volume += value
	speaker.Unlock()
}

func (t *BeepTrack) Unpause() {
	speaker.Lock()
	if t.Ctrl.Paused {
		t.Ctrl.Paused = false
	} else {
		log.Debug("tried to unpause while not paused")
	}
	speaker.Unlock()
}

func (t *BeepTrack) Pause() {
	speaker.Lock()
	if t.Ctrl.Paused {
		log.Debug("tried to pause while already paused")
	} else {
		t.Ctrl.Paused = true
	}
	speaker.Unlock()
}

func (t *BeepTrack) Close() {
	if t.ActiveStream != nil {
		log.Debug("closing activ stream")
		t.ActiveStream.Close()
		t.ActiveStream = nil
	}

	speaker.Close()

	if t.TempfileToCleanup != "" {
		log.Debug("Removing tempfile", "tmpfilename", t.TempfileToCleanup)
		err := os.Remove(t.TempfileToCleanup)
		if err != nil {
			log.Error("error cleaning up tempfile: ", t.TempfileToCleanup)
		}
	}
}

// Position returns the playback position in seconds
func (t *BeepTrack) Position() int {
	if t.Ctrl.Streamer == nil {
		log.Debug("streamer is not setup (nil), could not get position")
		return 0
	}

	streamer, ok := t.Ctrl.Streamer.(beep.StreamSeeker)
	if ok {
		position := t.SampleRate.D(streamer.Position())
		posSecs := position.Round(time.Second).Seconds()
		return int(posSecs)
	} else {
		log.Debug("streamer is no beep.StreamSeeker, could not get position")
		return 0
	}
}

// offset = pd.PlaybackQueue.Offset
func (t *BeepTrack) SetPosition(offset int) error {
	streamer, ok := t.Ctrl.Streamer.(beep.StreamSeeker)
	if ok {
		sampleRatePerSecond := t.SampleRate.N(time.Second)
		nextPosition := sampleRatePerSecond * offset
		log.Debug("SetPosition", "samplerate", sampleRatePerSecond, "nextPosition", nextPosition)
		return streamer.Seek(nextPosition)
	}
	return fmt.Errorf("streamer is not seekable")
}

func (t *BeepTrack) IsPlaying() bool {
	return t.Ctrl != nil && !t.Ctrl.Paused
}
