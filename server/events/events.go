package events

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type Event interface {
	Prepare(Event) string
}

type baseEvent struct {
	Name string `json:"name"`
}

func (e *baseEvent) Prepare(evt Event) string {
	str := strings.TrimPrefix(reflect.TypeOf(evt).String(), "*events.")
	e.Name = str[:0] + string(unicode.ToLower(rune(str[0]))) + str[1:]
	data, _ := json.Marshal(evt)
	return string(data)
}

type ScanStatus struct {
	baseEvent
	Scanning    bool  `json:"scanning"`
	Count       int64 `json:"count"`
	FolderCount int64 `json:"folderCount"`
}

type KeepAlive struct {
	baseEvent
	TS int64 `json:"ts"`
}

type ServerStart struct {
	baseEvent
	StartTime time.Time `json:"startTime"`
}
