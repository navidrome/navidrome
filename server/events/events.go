package events

type Event interface {
	EventName() string
}

type ScanStatus struct {
	Scanning bool  `json:"scanning"`
	Count    int64 `json:"count"`
}

func (s ScanStatus) EventName() string { return "scanStatus" }

type KeepAlive struct {
	TS int64 `json:"ts"`
}

func (s KeepAlive) EventName() string { return "keepAlive" }
