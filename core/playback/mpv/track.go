package mpv

import (
	"fmt"

	"github.com/DexterLB/mpvipc"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type MpvTrack struct {
	MediaFile    model.MediaFile
	PlaybackDone chan bool
}

func NewTrack(playbackDoneChannel chan bool, mf model.MediaFile) (*MpvTrack, error) {
	t := MpvTrack{}

	contentType := mf.ContentType()
	log.Debug("loading track", "trackname", mf.Path, "mediatype", contentType)

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

	err = conn.Set("pause", true)
	if err != nil {
		log.Fatal(err)
	}

	// save running stream for closing when switching tracks
	t.PlaybackDone = playbackDoneChannel
	t.MediaFile = mf

	return &t, nil
}

func (t *MpvTrack) String() string {
	return fmt.Sprintf("Name: %s", t.MediaFile.Path)
}

func (t *MpvTrack) SetVolume(value float64) {

}

func (t *MpvTrack) Unpause() {

}

func (t *MpvTrack) Pause() {

}

func (t *MpvTrack) Close() {

}

// Position returns the playback position in seconds
func (t *MpvTrack) Position() int {
	return 0
}

// offset = pd.PlaybackQueue.Offset
func (t *MpvTrack) SetPosition(offset int) error {
	return nil
}

func (t *MpvTrack) IsPlaying() bool {
	return false
}

func (t *MpvTrack) CloseDevice() {

}
