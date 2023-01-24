package subsonic

import (
	"net/http"

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

func parseAction(action string) ActionType {
	switch action {
	case "get":
		return ActionGet
	case "status":
		return ActionStatus
	case "set":
		return ActionSet
	case "start":
		return ActionStart
	case "stop":
		return ActionStop
	case "skip":
		return ActionSkip
	case "add":
		return ActionAdd
	case "clear":
		return ActionClear
	case "remove":
		return ActionRemove
	case "shuffle":
		return ActionShuffle
	case "setGain":
		return ActionSetGain
	default:
		return ActionUnknown
	}
}

func (api *Router) JukeboxControl(r *http.Request) (*responses.Subsonic, error) {
	actionString, err := requiredParamString(r, "action")
	if err != nil {
		return nil, err
	}

	action := parseAction(actionString)
	if action == ActionUnknown {
		return nil, newError(responses.ErrorMissingParameter, "Unknown action: %s", actionString)
	}

	return handleJukeboxAction(action, r)
}

func handleJukeboxAction(action ActionType, r *http.Request) (*responses.Subsonic, error) {
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
