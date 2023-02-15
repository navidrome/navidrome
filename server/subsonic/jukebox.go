package subsonic

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

type ActionType int

const (
	ActionUnknown ActionType = iota
	ActionGet
	ActionStatus
	ActionSet
	ActionStart
	ActionStop
	ActionSkip
	ActionAdd
	ActionClear
	ActionRemove
	ActionShuffle
	ActionSetGain
)

var ACTION_MAP = map[ActionType]string{
	ActionGet:     "get",
	ActionStatus:  "status",
	ActionSet:     "set",
	ActionStart:   "start",
	ActionStop:    "stop",
	ActionSkip:    "skip",
	ActionAdd:     "add",
	ActionClear:   "clear",
	ActionRemove:  "remove",
	ActionShuffle: "shuffle",
	ActionSetGain: "setGain",
}

func (action ActionType) String() string {
	value, found := ACTION_MAP[action]
	if found {
		return strings.ToUpper(value)
	}
	return "Unknown"
}

func parseAction(actionStr string) ActionType {
	for k, v := range ACTION_MAP {
		if v == actionStr {
			return k
		}
	}
	return ActionUnknown
}

func (api *Router) JukeboxControl(r *http.Request) (*responses.Subsonic, error) {
	user, err := requiredParamString(r, "u")
	if err != nil {
		return nil, err
	}

	actionString, err := requiredParamString(r, "action")
	if err != nil {
		return nil, err
	}

	pbServer := playback.GetInstance()
	pb, err := pbServer.GetDevice(user)
	if err != nil {
		return nil, err
	}

	action := parseAction(actionString)
	log.Debug(fmt.Sprintf("processing action: %s", action))

	switch action {
	case ActionGet:
		playlist, err := pb.Get(user)
		if err != nil {
			return nil, err
		}
		response := newResponse()
		response.JukeboxPlaylist = &playlist
		return response, nil
	case ActionStatus:
		status, err := pb.Status(user)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionSet:
		ids, err := requiredParamStrings(r, "id")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter id, err: %s", err)
		}
		status, err := pb.Set(user, ids)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionStart:
		status, err := pb.Start(user)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionStop:
		status, err := pb.Stop(user)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionSkip:
		index, err := requiredParamInt(r, "index")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter index, err: %s", err)
		}

		offset, err := requiredParamInt(r, "offset")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter offset, err: %s", err)
		}

		status, err := pb.Skip(user, index, offset)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionAdd:
		ids, err := requiredParamStrings(r, "id")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter id, err: %s", err)
		}

		status, err := pb.Add(user, ids)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionClear:
		status, err := pb.Clear(user)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionRemove:
		index, err := requiredParamInt(r, "index")
		if err != nil {
			return nil, err
		}

		status, err := pb.Remove(user, index)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionShuffle:
		status, err := pb.Shuffle(user)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionSetGain:
		gainStr, err := requiredParamString(r, "gain")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter gain, err: %s", err)
		}

		gain, err := strconv.ParseFloat(gainStr, 32)
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "error parsing gain integer value, err: %s", err)
		}
		status, err := pb.SetGain(user, float32(gain))
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionUnknown:
		return nil, newError(responses.ErrorMissingParameter, "Unknown action: %s", actionString)
	}

	return nil, newError(responses.ErrorMissingParameter, "action not found")
}

func statusResponse(status playback.DeviceStatus) *responses.Subsonic {
	response := newResponse()
	response.JukeboxStatus = &responses.JukeboxStatus{
		CurrentIndex: status.CurrentIndex,
		Playing:      status.Playing,
		Gain:         status.Gain,
		Position:     status.Position,
	}
	return response
}
