package mpv

// Audio-playback using mpv media-server. See mpv.io
// https://github.com/dexterlb/mpvipc
// https://mpv.io/manual/master/#json-ipc
// https://mpv.io/manual/master/#properties

import (
	"fmt"
	"os"
	"time"

	"github.com/dexterlb/mpvipc"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type MpvTrack struct {
	MediaFile     model.MediaFile
	PlaybackDone  chan bool
	Conn          *mpvipc.Connection
	IPCSocketName string
	Exe           *Executor
	CloseCalled   bool
}

func NewTrack(playbackDoneChannel chan bool, deviceName string, mf model.MediaFile) (*MpvTrack, error) {
	log.Debug("Loading track", "trackPath", mf.Path, "mediaType", mf.ContentType())

	if _, err := mpvCommand(); err != nil {
		return nil, err
	}

	tmpSocketName := TempFileName("mpv-ctrl-", ".socket")

	args := createMPVCommand(mpvComdTemplate, deviceName, mf.Path, tmpSocketName)
	exe, err := start(args)
	if err != nil {
		log.Error("Error starting mpv process", err)
		return nil, err
	}

	// wait for socket to show up
	err = waitForSocket(tmpSocketName, 3*time.Second, 100*time.Millisecond)
	if err != nil {
		log.Error("Error or timeout waiting for control socket", "socketname", tmpSocketName, err)
		return nil, err
	}

	conn := mpvipc.NewConnection(tmpSocketName)
	err = conn.Open()

	if err != nil {
		log.Error("Error opening new connection", err)
		return nil, err
	}

	theTrack := &MpvTrack{MediaFile: mf, PlaybackDone: playbackDoneChannel, Conn: conn, IPCSocketName: tmpSocketName, Exe: &exe, CloseCalled: false}

	go func() {
		conn.WaitUntilClosed()
		log.Info("Hitting end-of-stream, signalling on channel")
		if !theTrack.CloseCalled {
			playbackDoneChannel <- true
		}
	}()

	return theTrack, nil
}

func (t *MpvTrack) String() string {
	return fmt.Sprintf("Name: %s, Socket: %s", t.MediaFile.Path, t.IPCSocketName)
}

// Used to control the playback volume. A float value between 0.0 and 1.0.
func (t *MpvTrack) SetVolume(value float32) {
	// mpv's volume as described in the --volume parameter:
	// Set the startup volume. 0 means silence, 100 means no volume reduction or amplification.
	//  Negative values can be passed for compatibility, but are treated as 0.
	log.Debug("request for gain", "gain", value)
	vol := int(value * 100)

	err := t.Conn.Set("volume", vol)
	if err != nil {
		log.Error(err)
	}
	log.Debug("set volume", "volume", vol)
}

func (t *MpvTrack) Unpause() {
	err := t.Conn.Set("pause", false)
	if err != nil {
		log.Error(err)
	}
	log.Debug("Unpaused track", "track", t)
}

func (t *MpvTrack) Pause() {
	err := t.Conn.Set("pause", true)
	if err != nil {
		log.Error(err)
	}
	log.Info("paused track")
}

func (t *MpvTrack) Close() {
	log.Debug("closing resources")
	t.CloseCalled = true
	// trying to shutdown mpv process using socket
	if t.isSocketfilePresent() {
		log.Debug("sending shutdown command")
		_, err := t.Conn.Call("quit")
		if err != nil {
			log.Error("Error sending quit command to mpv-ipc socket", err)

			if t.Exe != nil {
				log.Debug("cancelling executor")
				err = t.Exe.Cancel()
				if err != nil {
					log.Error("error canceling executor")
				}
			}
		}
	}

	if t.isSocketfilePresent() {
		log.Debug("Removing socketfile", "socketfile", t.IPCSocketName)
		err := os.Remove(t.IPCSocketName)
		if err != nil {
			log.Error("error cleaning up socketfile: ", t.IPCSocketName)
		}
	}
}

func (t *MpvTrack) isSocketfilePresent() bool {
	if len(t.IPCSocketName) < 1 {
		return false
	}

	fileInfo, err := os.Stat(t.IPCSocketName)
	return err == nil && fileInfo != nil && !fileInfo.IsDir()
}

// Position returns the playback position in seconds.
// Every now and then the mpv IPC interface returns "mpv error: property unavailable"
// in this case we have to retry
func (t *MpvTrack) Position() int {
	retryCount := 0
	for {
		position, err := t.Conn.Get("time-pos")
		if err != nil && err.Error() == "mpv error: property unavailable" {
			retryCount += 1
			log.Debug("Got mpv error, retrying...", "retries", retryCount, err)
			if retryCount > 5 {
				return 0
			}
			break
		}

		if err != nil {
			log.Error("Error getting position in track", err)
			return 0
		}

		pos, ok := position.(float64)
		if !ok {
			log.Error("Could not cast position from mpv into float64", "position", position)
			return 0
		} else {
			return int(pos)
		}
	}
	return 0
}

func (t *MpvTrack) SetPosition(offset int) error {
	pos := t.Position()
	if pos == offset {
		log.Debug("no position difference, skipping operation")
		return nil
	}
	err := t.Conn.Set("time-pos", float64(offset))
	if err != nil {
		log.Error("could not set the position in track", "offset", offset, err)
		return err
	}
	log.Info("set position", "offset", offset)
	return nil
}

func (t *MpvTrack) IsPlaying() bool {
	pausing, err := t.Conn.Get("pause")
	if err != nil {
		log.Error("problem getting paused status", err)
		return false
	}

	pause, ok := pausing.(bool)
	if !ok {
		log.Error("could not cast pausing to boolean")
		return false
	}
	return !pause
}

func waitForSocket(path string, timeout time.Duration, pause time.Duration) error {
	start := time.Now()
	end := start.Add(timeout)
	var retries int = 0

	for {
		fileInfo, err := os.Stat(path)
		if err == nil && fileInfo != nil && !fileInfo.IsDir() {
			log.Debug("Socket found", "retries", retries, "waitTime", time.Since(start))
			return nil
		}
		if time.Now().After(end) {
			return fmt.Errorf("timeout reached: %s", timeout)
		}
		time.Sleep(pause)
		retries += 1
	}
}
