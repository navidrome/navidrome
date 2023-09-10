package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

const (
	ActionGet     = "get"
	ActionStatus  = "status"
	ActionSet     = "set"
	ActionStart   = "start"
	ActionStop    = "stop"
	ActionSkip    = "skip"
	ActionAdd     = "add"
	ActionClear   = "clear"
	ActionRemove  = "remove"
	ActionShuffle = "shuffle"
	ActionSetGain = "setGain"
)

func (api *Router) JukeboxControl(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	user := getUser(ctx)

	actionString, err := requiredParamString(r, "action")
	if err != nil {
		return nil, err
	}

	pbServer := playback.GetInstance()
	pb, err := pbServer.GetDeviceForUser(user.UserName)
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("processing action: %s", actionString))

	switch actionString {
	case ActionGet:
		mediafiles, status, err := pb.Get(ctx)
		if err != nil {
			return nil, err
		}

		playlist := responses.JukeboxPlaylist{
			JukeboxStatus: *deviceStatusToJukeboxStatus(status),
			Entry:         childrenFromMediaFiles(ctx, mediafiles),
		}

		response := newResponse()
		response.JukeboxPlaylist = &playlist
		return response, nil
	case ActionStatus:
		return createResponse(pb.Status(ctx))
	case ActionSet:
		ids, err := requiredParamStrings(r, "id")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter id, err: %s", err)
		}
		status, err := pb.Set(ctx, ids)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionStart:
		return createResponse(pb.Start(ctx))
	case ActionStop:
		return createResponse(pb.Stop(ctx))
	case ActionSkip:
		index, err := requiredParamInt(r, "index")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter index, err: %s", err)
		}

		offset, err := requiredParamInt(r, "offset")
		if err != nil {
			offset = 0
		}

		return createResponse(pb.Skip(ctx, index, offset))
	case ActionAdd:
		ids, err := requiredParamStrings(r, "id")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter id, err: %s", err)
		}

		return createResponse(pb.Add(ctx, ids))
	case ActionClear:
		return createResponse(pb.Clear(ctx))
	case ActionRemove:
		index, err := requiredParamInt(r, "index")
		if err != nil {
			return nil, err
		}

		return createResponse(pb.Remove(ctx, index))
	case ActionShuffle:
		return createResponse(pb.Shuffle(ctx))
	case ActionSetGain:
		gainStr, err := requiredParamString(r, "gain")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter gain, err: %s", err)
		}

		gain, err := strconv.ParseFloat(gainStr, 32)
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "error parsing gain integer value, err: %s", err)
		}

		return createResponse(pb.SetGain(ctx, float32(gain)))
	default:
		return nil, newError(responses.ErrorMissingParameter, "Unknown action: %s", actionString)
	}
}

// createResponse is to shorten the case-switch in the JukeboxController
func createResponse(status playback.DeviceStatus, err error) (*responses.Subsonic, error) {
	if err != nil {
		return nil, err
	}
	return statusResponse(status), nil
}

func statusResponse(status playback.DeviceStatus) *responses.Subsonic {
	response := newResponse()
	response.JukeboxStatus = deviceStatusToJukeboxStatus(status)
	return response
}

func deviceStatusToJukeboxStatus(status playback.DeviceStatus) *responses.JukeboxStatus {
	return &responses.JukeboxStatus{
		CurrentIndex: int32(status.CurrentIndex),
		Playing:      status.Playing,
		Gain:         status.Gain,
		Position:     int32(status.Position),
	}
}
