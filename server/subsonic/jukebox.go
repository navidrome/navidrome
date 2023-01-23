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

	response := newResponse()
	response.JukeboxStatus = &responses.JukeboxStatus{}
	response.JukeboxStatus.CurrentIndex = 0
	response.JukeboxStatus.Playing = false
	response.JukeboxStatus.Gain = 0
	response.JukeboxStatus.Position = 0
	return response, nil
}
