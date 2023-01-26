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

func (action ActionType) String() string {
	switch action {
	case ActionGet:
		return "Get"
	case ActionStatus:
		return "Status"
	case ActionSet:
		return "Set"
	case ActionStart:
		return "Start"
	case ActionStop:
		return "Stop"
	case ActionSkip:
		return "Skip"
	case ActionAdd:
		return "Add"
	case ActionClear:
		return "Clear"
	case ActionRemove:
		return "Remove"
	case ActionShuffle:
		return "Shuffle"
	case ActionSetGain:
		return "SetGain"
	default:
		return "Unknown"
	}
}

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
