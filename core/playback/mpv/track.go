package mpv

// Audio-playback using mpv media-server. See mpv.io
// https://github.com/dexterlb/mpvipc
// https://mpv.io/manual/master/#json-ipc
// https://mpv.io/manual/master/#properties

import (
	"fmt"
	"time"

	"github.com/DexterLB/mpvipc"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type MpvTrack struct {
	MediaFile    model.MediaFile
	PlaybackDone chan bool
	Conn         *mpvipc.Connection
}

func NewTrack(playbackDoneChannel chan bool, mf model.MediaFile) (*MpvTrack, error) {
	log.Debug("loading track", "trackname", mf.Path, "mediatype", mf.ContentType())

	if _, err := mpvCommand(); err != nil {
		return nil, err
	}

	tmpSocketName := TempFileName("mpv-ctrl-", ".socket")

	args := createMPVCommand(mpvComdTemplate, mf.Path, tmpSocketName)
	start(args)

	// FIXME: this won't stay like this ...
	time.Sleep(1000 * time.Millisecond)

	var err error

	conn := mpvipc.NewConnection(tmpSocketName)
	err = conn.Open()

	if err != nil {
		log.Error("error opening new connection", "error", err)
		return nil, err
	}

	go func() {
		conn.WaitUntilClosed()
		log.Info("Hitting end-of-stream, signalling on channel")
		playbackDoneChannel <- true
	}()

	return &MpvTrack{MediaFile: mf, PlaybackDone: playbackDoneChannel, Conn: conn}, nil
}

func (t *MpvTrack) String() string {
	return fmt.Sprintf("Name: %s", t.MediaFile.Path)
}

func (t *MpvTrack) SetVolume(value float64) {
	err := t.Conn.Set("volume", value)
	if err != nil {
		log.Error(err)
	}
	log.Info("set volume", "volume", value)
}

func (t *MpvTrack) Unpause() {
	err := t.Conn.Set("pause", false)
	if err != nil {
		log.Error(err)
	}
	log.Info("unpaused track")
}

func (t *MpvTrack) Pause() {
	err := t.Conn.Set("pause", true)
	if err != nil {
		log.Error(err)
	}
	log.Info("paused track")
}

func (t *MpvTrack) Close() {

}

// Position returns the playback position in seconds
func (t *MpvTrack) Position() int {
	position, err := t.Conn.Get("time-pos")
	if err != nil {
		log.Error("error getting position in track", "error", err)
		return 0
	}
	pos, ok := position.(float64)
	if !ok {
		log.Error("could not cast position from mpv into float64")
		return 0
	}
	return int(pos)
}

func (t *MpvTrack) SetPosition(offset int) error {
	err := t.Conn.Set("time-pos", float64(offset))
	if err != nil {
		log.Error("could not set the position in track", "offset", offset, "error", err)
		return err
	}
	log.Info("set position", "offset", offset)
	return nil
}

func (t *MpvTrack) IsPlaying() bool {
	pausing, err := t.Conn.Get("pause")
	if err != nil {
		log.Error("problem getting paused status", "error", err)
		return false
	}

	pause, ok := pausing.(bool)
	if !ok {
		log.Error("could not cast pausing to boolean")
		return false
	}
	return !pause
}