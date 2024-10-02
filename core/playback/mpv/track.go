package mpv

// Audio-playback using mpv media-server. See mpv.io
// https://github.com/dexterlb/mpvipc
// https://mpv.io/manual/master/#json-ipc
// https://mpv.io/manual/master/#properties

import (
	"context"
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

func NewTrack(ctx context.Context, playbackDoneChannel chan bool, deviceName string, mf model.MediaFile) (*MpvTrack, error) {
	log.Debug("Loading track", "trackPath", mf.Path, "mediaType", mf.ContentType())

	if _, err := mpvCommand(); err != nil {
		return nil, err
	}

	tmpSocketName := socketName("mpv-ctrl-", ".socket")

	args := createMPVCommand(deviceName, mf.Path, tmpSocketName)
	exe, err := start(ctx, args)
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
	log.Debug("Setting volume", "volume", value, "track", t)
	vol := int(value * 100)

	err := t.Conn.Set("volume", vol)
	if err != nil {
		log.Error("Error setting volume", "volume", value, "track", t, err)
	}
}

func (t *MpvTrack) Unpause() {
	log.Debug("Unpausing track", "track", t)
	err := t.Conn.Set("pause", false)
	if err != nil {
		log.Error("Error unpausing track", "track", t, err)
	}
}

func (t *MpvTrack) Pause() {
	log.Debug("Pausing track", "track", t)
	err := t.Conn.Set("pause", true)
	if err != nil {
		log.Error("Error pausing track", "track", t, err)
	}
}

func (t *MpvTrack) Close() {
	log.Debug("Closing resources", "track", t)
	t.CloseCalled = true
	// trying to shutdown mpv process using socket
	if t.isSocketFilePresent() {
		log.Debug("sending shutdown command")
		_, err := t.Conn.Call("quit")
		if err != nil {
			log.Warn("Error sending quit command to mpv-ipc socket", err)

			if t.Exe != nil {
				log.Debug("cancelling executor")
				err = t.Exe.Cancel()
				if err != nil {
					log.Warn("Error canceling executor", err)
				}
			}
		}
	}

	if t.isSocketFilePresent() {
		removeSocket(t.IPCSocketName)
	}
}

func (t *MpvTrack) isSocketFilePresent() bool {
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
			time.Sleep(time.Duration(retryCount) * time.Millisecond)
			continue
		}

		if err != nil {
			log.Error("Error getting position in track", "track", t, err)
			return 0
		}

		pos, ok := position.(float64)
		if !ok {
			log.Error("Could not cast position from mpv into float64", "position", position, "track", t)
			return 0
		} else {
			return int(pos)
		}
	}
}

func (t *MpvTrack) SetPosition(offset int) error {
	log.Debug("Setting position", "offset", offset, "track", t)
	pos := t.Position()
	if pos == offset {
		log.Debug("No position difference, skipping operation", "track", t)
		return nil
	}
	err := t.Conn.Set("time-pos", float64(offset))
	if err != nil {
		log.Error("Could not set the position in track", "track", t, "offset", offset, err)
		return err
	}
	return nil
}

func (t *MpvTrack) IsPlaying() bool {
	log.Debug("Checking if track is playing", "track", t)
	pausing, err := t.Conn.Get("pause")
	if err != nil {
		log.Error("Problem getting paused status", "track", t, err)
		return false
	}

	pause, ok := pausing.(bool)
	if !ok {
		log.Error("Could not cast pausing to boolean", "track", t, "value", pausing)
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
