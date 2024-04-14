//go:build !windows

package mpv

import (
	"os"

	"github.com/navidrome/navidrome/log"
)

func (t *MpvTrack) Close() {
	log.Debug("Closing resources", "track", t)
	t.CloseCalled = true
	// trying to shutdown mpv process using socket
	if t.isSocketFilePresent() {
		log.Debug("sending shutdown command")
		_, err := t.Conn.Call("quit")
		if err != nil {
			log.Error("Error sending quit command to mpv-ipc socket", err)

			if t.Exe != nil {
				log.Debug("cancelling executor")
				err = t.Exe.Cancel()
				if err != nil {
					log.Error("Error canceling executor", err)
				}
			}
		}
	}

	if t.isSocketFilePresent() {
		log.Debug("Removing socketfile", "socketfile", t.IPCSocketName)
		err := os.Remove(t.IPCSocketName)
		if err != nil {
			log.Error("Error cleaning up socketfile", "socketfile", t.IPCSocketName, err)
		}
	}
}
