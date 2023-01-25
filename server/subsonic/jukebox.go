package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/log"
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

	action := parseAction(actionString)
	if action == ActionUnknown {
		return nil, newError(responses.ErrorMissingParameter, "Unknown action: %s", actionString)
	}

	parameter, err := parseActionParameter(action, r)
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	return handleJukeboxAction(ctx, action, user, parameter)
}

func parseActionParameter(action ActionType, r *http.Request) (ActionParameter, error) {
	switch action {
	case ActionRemove:
		index, err := getParameterAsInt64(r, "index")
		if err != nil {
			return ActionParameter{}, err
		}

		return ActionParameter{Index: index}, nil
	case ActionSkip:
		index, err := getParameterAsInt64(r, "index")
		if err != nil {
			return ActionParameter{}, err
		}

		offset, err := getParameterAsInt64(r, "offset")
		if err != nil {
			return ActionParameter{}, err
		}

		return ActionParameter{Index: index, Offset: offset}, nil
	case ActionAdd, ActionSet:
		id, err := requiredParamString(r, "id")
		if err != nil {
			return ActionParameter{}, newError(responses.ErrorMissingParameter, "missing parameter id, err: %w", err)
		}
		return ActionParameter{Id: id}, nil

	case ActionSetGain:
		gainStr, err := requiredParamString(r, "gain")
		if err != nil {
			return ActionParameter{}, newError(responses.ErrorMissingParameter, "missing parameter gain, err: %w", err)
		}

		gain, err := strconv.ParseFloat(gainStr, 64)
		if err != nil {
			return ActionParameter{}, newError(responses.ErrorMissingParameter, "error parsing gain integer value, err: %w", err)
		}
		return ActionParameter{Gain: gain}, nil
	}

	return ActionParameter{}, nil
}

func getParameterAsInt64(r *http.Request, name string) (int64, error) {
	indexStr, err := requiredParamString(r, name)
	if err != nil {
		return 0, newError(responses.ErrorMissingParameter, "missing parameter %s, err: %w", name, err)
	}

	index, err := strconv.ParseInt(indexStr, 10, 64)
	if err != nil {
		return 0, newError(responses.ErrorMissingParameter, "error parsing %s integer value, err: %w", name, err)
	}
	return index, nil
}

func handleJukeboxAction(ctx context.Context, action ActionType, user string, parameter ActionParameter) (*responses.Subsonic, error) {
	log.Debug(fmt.Sprintf("Handle action: %s for user: %s, parameter: %v", action, user, parameter))
	playback := playback.GetInstance()
	playback.Play()

	response := createJukeboxStatus(0, false, 0, 0)
	return response, nil
}

func createJukeboxStatus(currentIndex int64, playing bool, gain float64, position int64) *responses.Subsonic {
	response := newResponse()
	response.JukeboxStatus = &responses.JukeboxStatus{
		CurrentIndex: currentIndex,
		Playing:      playing,
		Gain:         gain,
		Position:     position,
	}
	return response
}
