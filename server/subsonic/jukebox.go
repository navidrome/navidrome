package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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
	pb, err := pbServer.GetDeviceForUser(user)
	if err != nil {
		return nil, err
	}

	action := parseAction(actionString)
	log.Debug(fmt.Sprintf("processing action: %s", action))

	switch action {
	case ActionGet:
		mediafiles, status, err := pb.Get(r.Context())
		if err != nil {
			return nil, err
		}

		playlist := responses.JukeboxPlaylist{
			JukeboxStatus: *deviceStatusToJukeboxStatus(status),
			Entry:         mediafilesToChilds(r.Context(), mediafiles),
		}

		response := newResponse()
		response.JukeboxPlaylist = &playlist
		return response, nil
	case ActionStatus:
		return createResponse(pb.Status(r.Context()))
	case ActionSet:
		ids, err := requiredParamStrings(r, "id")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter id, err: %s", err)
		}
		status, err := pb.Set(r.Context(), ids)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionStart:
		return createResponse(pb.Start(r.Context()))
	case ActionStop:
		return createResponse(pb.Stop(r.Context()))
	case ActionSkip:
		index, err := requiredParamInt(r, "index")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter index, err: %s", err)
		}

		offset, err := requiredParamInt(r, "offset")
		if err != nil {
			offset = 0
		}

		return createResponse(pb.Skip(r.Context(), index, offset))
	case ActionAdd:
		ids, err := requiredParamStrings(r, "id")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter id, err: %s", err)
		}

		return createResponse(pb.Add(r.Context(), ids))
	case ActionClear:
		return createResponse(pb.Clear(r.Context()))
	case ActionRemove:
		index, err := requiredParamInt(r, "index")
		if err != nil {
			return nil, err
		}

		return createResponse(pb.Remove(r.Context(), index))
	case ActionShuffle:
		return createResponse(pb.Shuffle(r.Context()))
	case ActionSetGain:
		gainStr, err := requiredParamString(r, "gain")
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter gain, err: %s", err)
		}

		gain, err := strconv.ParseFloat(gainStr, 32)
		if err != nil {
			return nil, newError(responses.ErrorMissingParameter, "error parsing gain integer value, err: %s", err)
		}

		return createResponse(pb.SetGain(r.Context(), float32(gain)))
	case ActionUnknown:
		return nil, newError(responses.ErrorMissingParameter, "Unknown action: %s", actionString)
	}

	return nil, newError(responses.ErrorMissingParameter, "action not found")
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
		CurrentIndex: status.CurrentIndex,
		Playing:      status.Playing,
		Gain:         status.Gain,
		Position:     status.Position,
	}
}

func mediafilesToChilds(ctx context.Context, items model.MediaFiles) []responses.Child {
	result := []responses.Child{}
	for _, item := range items {
		result = append(result, ChildFromMediaFile(ctx, item))
	}
	return result
}
