//go:build windows

package mpv

func (t *MpvTrack) Close() {
	// Windows automatically handles closing
	// and cleaning up named pipe
}
