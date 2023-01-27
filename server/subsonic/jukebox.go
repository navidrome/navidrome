package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

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

	switch parseAction(actionString) {
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
		id, err := requiredParamString(r, "id")
		if err != nil {
			return newFailure(), newError(responses.ErrorMissingParameter, "missing parameter id, err: %s", err)
		}
		status, err := pb.Set(user, id)
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
		index, err := getParameterAsInt64(r, "index")
		if err != nil {
			return newFailure(), newError(responses.ErrorMissingParameter, "missing parameter index, err: %s", err)
		}

		offset, err := getParameterAsInt64(r, "offset")
		if err != nil {
			return newFailure(), newError(responses.ErrorMissingParameter, "missing parameter offset, err: %s", err)
		}

		status, err := pb.Skip(user, index, offset)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionAdd:
		id, err := requiredParamString(r, "id")
		if err != nil {
			return newFailure(), newError(responses.ErrorMissingParameter, "missing parameter id, err: %s", err)
		}
		status, err := pb.Add(user, id)
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
		index, err := getParameterAsInt64(r, "index")
		if err != nil {
			return newFailure(), err
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
			return newFailure(), newError(responses.ErrorMissingParameter, "missing parameter gain, err: %s", err)
		}

		gain, err := strconv.ParseFloat(gainStr, 64)
		if err != nil {
			return newFailure(), newError(responses.ErrorMissingParameter, "error parsing gain integer value, err: %s", err)
		}
		status, err := pb.SetGain(user, gain)
		if err != nil {
			return nil, err
		}
		return statusResponse(status), nil
	case ActionUnknown:
		return nil, newError(responses.ErrorMissingParameter, "Unknown action: %s", actionString)
	}

	return nil, newError(responses.ErrorMissingParameter, "action not found")
}

func statusResponse(status responses.JukeboxStatus) *responses.Subsonic {
	response := newResponse()
	response.JukeboxStatus = &status
	return response
}

func getParameterAsInt64(r *http.Request, name string) (int64, error) {
	indexStr, err := requiredParamString(r, name)
	if err != nil {
		return 0, fmt.Errorf("missing parameter %s, err: %s", name, err)
	}

	index, err := strconv.ParseInt(indexStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing %s integer value, err: %s", name, err)
	}
	return index, nil
}
