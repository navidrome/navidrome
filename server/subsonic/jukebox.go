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

	pb := playback.GetInstance()

	switch parseAction(actionString) {
	case ActionGet:
		pb.Get(user)
		return newResponse(), nil
	case ActionStatus:
		pb.Status(user)
		return newResponse(), nil
	case ActionSet:
		id, err := requiredParamString(r, "id")
		if err != nil {
			return newFailure(), fmt.Errorf("missing parameter id, err: %s", err)
		}
		pb.Set(user, id)
		return newResponse(), nil
	case ActionStart:
		pb.Start(user)
		return newResponse(), nil
	case ActionStop:
		pb.Stop(user)
		return newResponse(), nil
	case ActionSkip:
		index, err := getParameterAsInt64(r, "index")
		if err != nil {
			return newFailure(), err
		}

		offset, err := getParameterAsInt64(r, "offset")
		if err != nil {
			return newFailure(), err
		}

		pb.Skip(user, index, offset)
		return newResponse(), nil
	case ActionAdd:
		id, err := requiredParamString(r, "id")
		if err != nil {
			return newFailure(), fmt.Errorf("missing parameter id, err: %s", err)
		}
		pb.Add(user, id)
		return newResponse(), nil
	case ActionClear:
		pb.Clear(user)
		return newResponse(), nil
	case ActionRemove:
		index, err := getParameterAsInt64(r, "index")
		if err != nil {
			return newFailure(), err
		}

		pb.Remove(user, index)
		return newResponse(), nil
	case ActionShuffle:
		pb.Shuffle(user)
		return newResponse(), nil
	case ActionSetGain:
		gainStr, err := requiredParamString(r, "gain")
		if err != nil {
			return newFailure(), fmt.Errorf("missing parameter gain, err: %s", err)
		}

		gain, err := strconv.ParseFloat(gainStr, 64)
		if err != nil {
			return newFailure(), fmt.Errorf("error parsing gain integer value, err: %s", err)
		}
		pb.SetGain(user, gain)
		return newResponse(), nil
	case ActionUnknown:
		return nil, newError(responses.ErrorMissingParameter, "Unknown action: %s", actionString)
	}

	return nil, newError(responses.ErrorMissingParameter, "action not found")
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
