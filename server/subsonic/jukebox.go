package subsonic

import (
	"context"
	"fmt"
	"net/http"

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
