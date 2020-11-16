package events

import "time"

type Event interface {
	EventName() string
}

type ScanStatus struct {
	Scanning    bool  `json:"scanning"`
	Count       int64 `json:"count"`
	FolderCount int64 `json:"folderCount"`
}

func (s ScanStatus) EventName() string { return "scanStatus" }

type KeepAlive struct {
	TS int64 `json:"ts"`
}

func (s KeepAlive) EventName() string { return "keepAlive" }

type ServerStart struct {
	StartTime time.Time `json:"startTime"`
}

func (s ServerStart) EventName() string { return "serverStart" }
