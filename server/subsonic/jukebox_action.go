package subsonic

import "strings"

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
