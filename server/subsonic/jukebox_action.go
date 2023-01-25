package subsonic

type ActionParameter struct {
	Index  int64
	Offset int64
	Id     string
	Gain   float64
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
