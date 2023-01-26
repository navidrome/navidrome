package subsonic

import (
	"fmt"
	"net/http"
	"strconv"
)

type Action struct {
	actionType ActionType
	user       string
	Index      int64
	Offset     int64
	Id         string
	Gain       float64
}

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
		return value
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

func parseActionParameter(action ActionType, r *http.Request) (Action, error) {
	switch action {
	case ActionRemove:
		index, err := getParameterAsInt64(r, "index")
		if err != nil {
			return Action{}, err
		}

		return Action{Index: index}, nil
	case ActionSkip:
		index, err := getParameterAsInt64(r, "index")
		if err != nil {
			return Action{}, err
		}

		offset, err := getParameterAsInt64(r, "offset")
		if err != nil {
			return Action{}, err
		}

		return Action{Index: index, Offset: offset}, nil
	case ActionAdd, ActionSet:
		id, err := requiredParamString(r, "id")
		if err != nil {
			return Action{}, fmt.Errorf("missing parameter id, err: %s", err)
		}
		return Action{Id: id}, nil

	case ActionSetGain:
		gainStr, err := requiredParamString(r, "gain")
		if err != nil {
			return Action{}, fmt.Errorf("missing parameter gain, err: %s", err)
		}

		gain, err := strconv.ParseFloat(gainStr, 64)
		if err != nil {
			return Action{}, fmt.Errorf("error parsing gain integer value, err: %s", err)
		}
		return Action{Gain: gain}, nil
	}

	return Action{}, nil
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
