package mpv

// Audio-playback using mpv media-server. See mpv.io
// https://github.com/dexterlb/mpvipc
// https://mpv.io/manual/master/#json-ipc
// https://mpv.io/manual/master/#properties

import (
	"fmt"

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

	args := createMPVCommand(mpvComdTemplate, mf.Path, mpvSocket)
	start(args)

	var err error

	conn := mpvipc.NewConnection(mpvSocket)
	err = conn.Open()

	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &MpvTrack{MediaFile: mf, PlaybackDone: playbackDoneChannel, Conn: conn}, nil
}

func (t *MpvTrack) String() string {
	return fmt.Sprintf("Name: %s", t.MediaFile.Path)
}

func (t *MpvTrack) SetVolume(value float64) {
	err := t.Conn.Set("volume", value)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("set volume", "volume", value)
}

func (t *MpvTrack) Unpause() {
	err := t.Conn.Set("pause", false)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("unpaused track")
}

func (t *MpvTrack) Pause() {
	err := t.Conn.Set("pause", true)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("paused track")
}

func (t *MpvTrack) Close() {

}

// Position returns the playback position in seconds
func (t *MpvTrack) Position() int {
	position, err := t.Conn.Get("time-pos")
	if err != nil {
		log.Fatal(err)
		return 0
	}
	pos, ok := position.(int)
	if !ok {
		return 0
	}
	return pos
}

// offset = pd.PlaybackQueue.Offset
func (t *MpvTrack) SetPosition(offset int) error {
	err := t.Conn.Set("time-pos", offset)
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Info("set position", "offset", offset)
	return nil
}

func (t *MpvTrack) IsPlaying() bool {
	pausing, err := t.Conn.Get("pause")
	if err != nil {
		log.Fatal("problem getting paused status", "error", err)
		return false
	}

	pause, ok := pausing.(bool)
	if !ok {
		return false
	}
	return pause
}

func (t *MpvTrack) CloseDevice() {

}
