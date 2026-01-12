// Package playback implements audio playback using PlaybackDevices. It is used to implement the Jukebox mode in turn.
// It makes use of the MPV library to do the playback. Major parts are:
// - decoder which includes decoding and transcoding of various audio file formats
// - device implementing the basic functions to work with audio devices like set, play, stop, skip, ...
// - queue a simple playlist
package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/playback/mpv"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/singleton"
)

// Define a Go struct that mirrors the data structure returned by the Lua script.
type PlaylistTrack struct {
	// Basic info from mpv's playlist property
	Filename      string `json:"filename"`
	IsPlaying     bool   `json:"isPlaying"`
	IsCurrent     bool   `json:"isCurrent"`
	PlaylistIndex int    `json:"playlistIndex"`

	// Rich metadata from our Lua cache (optional fields)
	Title      string  `json:"title,omitempty"`
	Artist     string  `json:"artist,omitempty"`
	Album      string  `json:"album,omitempty"`
	Year       string  `json:"year,omitempty"`
	Genre      string  `json:"genre,omitempty"`
	Track      int     `json:"track,omitempty"`
	DiscNumber int     `json:"discNumber,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
	Size       float64 `json:"size,omitempty"`
	Path       string  `json:"path,omitempty"`
	Suffix     string  `json:"suffix,omitempty"`
	BitRate    float64 `json:"bitRate,omitempty"`
}

type PlaybackServer interface {
	Run(ctx context.Context) error
	GetDeviceForUser(user string) (*playbackDevice, error)
	GetMediaFile(id string) (*model.MediaFile, error)
	GetConnection() (*mpv.MpvConnection, error)
	LoadFile(mf *model.MediaFile, append bool, playNow bool) (bool, error)
	Clear() (bool, error)
	Remove(index int) (bool, error)
	Shuffle() (bool, error)
	SetGain(gain float32) (bool, error)
	IsPlaying() bool
	Skip(index int, offset int) (bool, error)
	Stop() (bool, error)
	Start() (bool, error)
	SetPosition(offset int) error
	Position() int
	GetPlaylistPosition() int
	GetPlaylistIDs() []string
}

type playbackServer struct {
	ctx             *context.Context
	datastore       model.DataStore
	playbackDevices []playbackDevice
	Conn            *mpv.MpvConnection
}

// GetInstance returns the playback-server singleton
func GetInstance(ds model.DataStore) PlaybackServer {
	return singleton.GetInstance(func() *playbackServer {
		conn, err := mpv.NewConnection(context.Background(), "auto")
		if err != nil {
			log.Error("Error opening new connection", err)
			return nil
		}
		return &playbackServer{datastore: ds, Conn: conn}
	})
}

// Run starts the playback server which serves request until canceled using the given context
func (ps *playbackServer) Run(ctx context.Context) error {
	ps.ctx = &ctx

	devices, err := ps.initDeviceStatus(ctx, conf.Server.Jukebox.Devices, conf.Server.Jukebox.Default)
	if err != nil {
		return err
	}
	ps.playbackDevices = devices
	log.Info(ctx, fmt.Sprintf("%d audio devices found", len(devices)))

	defaultDevice, _ := ps.getDefaultDevice()

	log.Info(ctx, "Using audio device: "+defaultDevice.DeviceName)

	<-ctx.Done()

	// Should confirm all subprocess are terminated before returning
	return nil
}

func (ps *playbackServer) GetConnection() (*mpv.MpvConnection, error) {
	log.Debug("Returning connection")
	return ps.Conn, nil
}

func (ps *playbackServer) initDeviceStatus(ctx context.Context, devices []conf.AudioDeviceDefinition, defaultDevice string) ([]playbackDevice, error) {
	pbDevices := make([]playbackDevice, max(1, len(devices)))
	defaultDeviceFound := false

	if defaultDevice == "" {
		// if there are no devices given and no default device, we create a synthetic device named "auto"
		if len(devices) == 0 {
			pbDevices[0] = *NewPlaybackDevice(ctx, ps, "auto", "auto")
		}

		// if there is but only one entry and no default given, just use that.
		if len(devices) == 1 {
			if len(devices[0]) != 2 {
				return []playbackDevice{}, fmt.Errorf("audio device definition ought to contain 2 fields, found: %d ", len(devices[0]))
			}
			pbDevices[0] = *NewPlaybackDevice(ctx, ps, devices[0][0], devices[0][1])
		}

		if len(devices) > 1 {
			return []playbackDevice{}, fmt.Errorf("number of audio device found is %d, but no default device defined. Set Jukebox.Default", len(devices))
		}

		pbDevices[0].Default = true
		return pbDevices, nil
	}

	for idx, audioDevice := range devices {
		if len(audioDevice) != 2 {
			return []playbackDevice{}, fmt.Errorf("audio device definition ought to contain 2 fields, found: %d ", len(audioDevice))
		}

		pbDevices[idx] = *NewPlaybackDevice(ctx, ps, audioDevice[0], audioDevice[1])

		if audioDevice[0] == defaultDevice {
			pbDevices[idx].Default = true
			defaultDeviceFound = true
		}
	}

	if !defaultDeviceFound {
		return []playbackDevice{}, fmt.Errorf("default device name not found: %s ", defaultDevice)
	}
	return pbDevices, nil
}

func (ps *playbackServer) getDefaultDevice() (*playbackDevice, error) {
	for idx := range ps.playbackDevices {
		if ps.playbackDevices[idx].Default {
			return &ps.playbackDevices[idx], nil
		}
	}
	return nil, fmt.Errorf("no default device found")
}

// GetMediaFile retrieves the MediaFile given by the id parameter
func (ps *playbackServer) GetMediaFile(id string) (*model.MediaFile, error) {
	return ps.datastore.MediaFile(*ps.ctx).Get(id)
}

// GetDeviceForUser returns the audio playback device for the given user. As of now this is but only the default device.
func (ps *playbackServer) GetDeviceForUser(user string) (*playbackDevice, error) {
	log.Debug("Processing GetDevice", "user", user)
	// README: here we might plug-in the user-device mapping one fine day
	device, err := ps.getDefaultDevice()
	if err != nil {
		return nil, err
	}
	device.User = user
	return device, nil
}

// LoadFile loads a file into the MPV player, and optionally plays it right away
func (ps *playbackServer) LoadFile(mf *model.MediaFile, append bool, playNow bool) (bool, error) {
	log.Debug("Loading file", "mf", mf, "append", append, "playNow", playNow)
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false, err
	}

	command := ""
	if append {
		command += "append"
		if playNow {
			log.Debug("Stopping current file")
			_, err := conn.Conn.Call("playlist-play-index", "none")
			if err != nil {
				log.Error("Error stopping current file", "mf", mf, err)
			}
			command += "-play"
		}
	} else {
		command += "replace"
	}

	log.Debug("Loading file", "mf", mf.AbsolutePath(), "command", command)
	// Example: Tell Lua about the track's ID
	_, err = conn.Conn.Call("script-message", "attach-id", mf.AbsolutePath(), mf.ID)
	if err != nil {
		log.Error("Error attaching ID", "mf", mf, err)
		return false, err
	}

	_, err = conn.Conn.Call("loadfile", mf.AbsolutePath(), command)
	if err != nil {
		log.Error("Error loading file", "mf", mf, err)
		return false, err
	}
	return true, nil
}

// Clear clears the playlist
func (ps *playbackServer) Clear() (bool, error) {
	log.Debug("Clearing playlist")
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false, err
	}
	_, err = conn.Conn.Call("stop")
	if err != nil {
		log.Error("Error stopping current file", err)
		return false, err
	}
	return true, nil
}

// Remove the playlist entry at the given index. Index values start counting with 0.
// The special value current removes the current entry.
// Note that removing the current entry also stops playback and starts playing the next entry.
func (ps *playbackServer) Remove(index int) (bool, error) {
	log.Debug("Removing file", "index", index)
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false, err
	}
	_, err = conn.Conn.Call("playlist-remove", index)
	if err != nil {
		log.Error("Error removing file", "index", index, err)
		return false, err
	}
	return true, nil
}

// SetGain is used to control the playback volume. A float value between 0.0 and 1.0.
func (ps *playbackServer) SetGain(gain float32) (bool, error) {
	// mpv's volume as described in the --volume parameter:
	// Set the startup volume. 0 means silence, 100 means no volume reduction or amplification.
	//  Negative values can be passed for compatibility, but are treated as 0.
	vol := int(gain * 100)
	log.Debug("Setting volume", "volume", vol)
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false, err
	}
	err = conn.Conn.Set("volume", vol)
	if err != nil {
		log.Error("Error setting volume", "volume", vol, err)
		return false, err
	}
	return true, nil
}

// Shuffle shuffles the playlist
func (ps *playbackServer) Shuffle() (bool, error) {
	log.Debug("Shuffling playlist")
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false, err
	}
	_, err = conn.Conn.Call("playlist-shuffle")
	if err != nil {
		log.Error("Error shuffling playlist", err)
		return false, err
	}
	return true, nil
}

// IsPlaying checks if the currently playing track is paused
func (ps *playbackServer) IsPlaying() bool {
	log.Debug("Checking if track is playing")
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false
	}
	pausing, err := conn.Conn.Get("pause")
	if err != nil {
		log.Error("Problem getting paused status", err)
		return false
	}

	pause, ok := pausing.(bool)
	if !ok {
		log.Error("Could not cast pausing to boolean", "value", pausing)
		return false
	}
	log.Debug("Checked if track is playing", "pausing", pause)
	return !pause
}

// Skip skips to the given track
func (ps *playbackServer) Skip(index int, offset int) (bool, error) {
	log.Debug("Skipping to track", "index", index, "offset", offset)
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false, err
	}
	_, err = conn.Conn.Call("playlist-play-index", index)
	if err != nil {
		log.Error("Error skipping to track", "index", index, err)
		return false, err
	}
	if offset > 0 {
		_, err = conn.Conn.Call("seek", offset, "absolute")
		if err != nil {
			log.Error("Error skipping to offset", "offset", offset, err)
			return false, err
		}
	}
	return true, nil
}

// Stop stops the currently playing track
func (ps *playbackServer) Stop() (bool, error) {
	log.Debug("Stopping track")
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false, err
	}
	err = conn.Conn.Set("pause", true)
	if err != nil {
		log.Error("Error stopping track", "err", err)
		return false, err
	}
	return true, nil
}

// Start starts the currently playing track
func (ps *playbackServer) Start() (bool, error) {
	log.Debug("Starting track")
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return false, err
	}
	err = conn.Conn.Set("pause", false)
	if err != nil {
		log.Error("Error starting track", "err", err)
		return false, err
	}
	return true, nil
}

// Position returns the playback position in seconds.
// Every now and then the mpv IPC interface returns "mpv error: property unavailable"
// in this case we have to retry
func (ps *playbackServer) Position() int {
	retryCount := 0
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return 0
	}
	for {
		position, err := conn.Conn.Get("time-pos")
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
}

// SetPosition sets the position in the currently playing track
func (ps *playbackServer) SetPosition(offset int) error {
	log.Debug("Setting position", "offset", offset)
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return err
	}
	_, err = conn.Conn.Call("seek", float64(offset), "absolute")
	if err != nil {
		log.Error("Error setting position", "offset", offset, err)
		return err
	}
	return nil
}

// Current position on playlist. The first entry is on position 0.
func (ps *playbackServer) GetPlaylistPosition() int {
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting connection", err)
		return 0
	}
	position, err := conn.Conn.Get("playlist-pos")
	if err != nil {
		log.Error("Error getting current position", err)
		return 0
	}
	return int(position.(float64))
}

// This function now retrieves an ordered list of database IDs from the mpv playlist.
// The return type has been changed from []PlaylistTrack to []string.
func (ps *playbackServer) GetPlaylistIDs() []string {
	conn, err := ps.GetConnection()
	if err != nil {
		log.Error("Error getting mpv connection", "error", err)
		return []string{}
	}

	// Call the new, renamed script message
	_, err = conn.Conn.Call("script-message", "update_playlist_ids_property")
	if err != nil {
		log.Error("Error calling mpv script 'update_playlist_ids_property'", "error", err)
		return []string{}
	}

	// Poll for the property with a retry loop to avoid a race condition
	var result interface{}
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		// Use the new, renamed property. Using GetProperty is slightly safer.
		result, err = conn.Conn.Get("user-data/ext-playlist-ids")
		if err == nil && result != nil {
			break // Success!
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		log.Error("Error getting 'user-data/ext-playlist-ids' property after retries", "error", err)
		return []string{}
	}
	if result == nil {
		log.Error("Property 'user-data/ext-playlist-ids' was still nil after retries.")
		return []string{}
	}

	// The result from mpv is the JSON string of the ID list.
	jsonString, ok := result.(string)
	if !ok {
		log.Error("Expected a string from 'ext-playlist-ids' property but got something else", "type", fmt.Sprintf("%T", result))
		return []string{}
	}

	// Unmarshal the JSON array of strings into a slice of strings.
	var idList []string
	if err := json.Unmarshal([]byte(jsonString), &idList); err != nil {
		log.Error("Error unmarshalling playlist ID JSON into slice of strings", "error", err, "data", jsonString)
		return []string{}
	}

	log.Info("Successfully retrieved playlist IDs from mpv.", "id_count", len(idList))
	return idList
}
