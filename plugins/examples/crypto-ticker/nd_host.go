// Host function wrappers for Navidrome plugin services.
// These allow the plugin to call host functions provided by Navidrome.
package main

import (
	"encoding/json"
	"errors"

	pdk "github.com/extism/go-pdk"
)

// WebSocket host functions

//go:wasmimport extism:host/user websocket_connect
func websocket_connect(uint64) uint64

//go:wasmimport extism:host/user websocket_sendtext
func websocket_sendtext(connectionID uint64, message uint64) uint64

//go:wasmimport extism:host/user websocket_closeconnection
func websocket_closeconnection(connectionID uint64, code int32, reason uint64) uint64

// WebSocketConnectRequest is the request type for WebSocket.Connect
type WebSocketConnectRequest struct {
	Url          string            `json:"url"`
	Headers      map[string]string `json:"headers,omitempty"`
	ConnectionID string            `json:"connectionID,omitempty"`
}

// WebSocketConnectResponse is the response type for WebSocket.Connect
type WebSocketConnectResponse struct {
	NewConnectionID string `json:"newConnectionID,omitempty"`
	Error           string `json:"error,omitempty"`
}

// WebSocketConnect establishes a WebSocket connection to the specified URL.
func WebSocketConnect(url string, headers map[string]string, connectionID string) (string, error) {
	req := WebSocketConnectRequest{
		Url:          url,
		Headers:      headers,
		ConnectionID: connectionID,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	reqMem := pdk.AllocateBytes(reqBytes)
	defer reqMem.Free()

	responsePtr := websocket_connect(reqMem.Offset())
	responseMem := pdk.FindMemory(responsePtr)
	responseBytes := responseMem.ReadBytes()

	var resp WebSocketConnectResponse
	if err := json.Unmarshal(responseBytes, &resp); err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.NewConnectionID, nil
}

// WebSocketSendText sends a text message over an established WebSocket connection.
func WebSocketSendText(connectionID, message string) error {
	connMem := pdk.AllocateString(connectionID)
	defer connMem.Free()
	msgMem := pdk.AllocateString(message)
	defer msgMem.Free()

	responsePtr := websocket_sendtext(connMem.Offset(), msgMem.Offset())
	if responsePtr != 0 {
		responseMem := pdk.FindMemory(responsePtr)
		errStr := string(responseMem.ReadBytes())
		if errStr != "" {
			return errors.New(errStr)
		}
	}
	return nil
}

// WebSocketCloseConnection gracefully closes a WebSocket connection.
func WebSocketCloseConnection(connectionID string, code int32, reason string) error {
	connMem := pdk.AllocateString(connectionID)
	defer connMem.Free()
	reasonMem := pdk.AllocateString(reason)
	defer reasonMem.Free()

	responsePtr := websocket_closeconnection(connMem.Offset(), code, reasonMem.Offset())
	if responsePtr != 0 {
		responseMem := pdk.FindMemory(responsePtr)
		errStr := string(responseMem.ReadBytes())
		if errStr != "" {
			return errors.New(errStr)
		}
	}
	return nil
}

// Scheduler host functions

//go:wasmimport extism:host/user scheduler_scheduleonetime
func scheduler_scheduleonetime(delaySeconds int32, payload uint64, scheduleID uint64) uint64

//go:wasmimport extism:host/user scheduler_schedulerecurring
func scheduler_schedulerecurring(cronExpression uint64, payload uint64, scheduleID uint64) uint64

//go:wasmimport extism:host/user scheduler_cancelschedule
func scheduler_cancelschedule(scheduleID uint64) uint64

// SchedulerScheduleOneTimeResponse is the response type for Scheduler.ScheduleOneTime
type SchedulerScheduleOneTimeResponse struct {
	NewScheduleID string `json:"newScheduleID,omitempty"`
	Error         string `json:"error,omitempty"`
}

// SchedulerScheduleRecurringResponse is the response type for Scheduler.ScheduleRecurring
type SchedulerScheduleRecurringResponse struct {
	NewScheduleID string `json:"newScheduleID,omitempty"`
	Error         string `json:"error,omitempty"`
}

// SchedulerScheduleOneTime schedules a one-time task to run after delaySeconds.
func SchedulerScheduleOneTime(delaySeconds int32, payload, scheduleID string) (string, error) {
	payloadMem := pdk.AllocateString(payload)
	defer payloadMem.Free()
	scheduleIDMem := pdk.AllocateString(scheduleID)
	defer scheduleIDMem.Free()

	responsePtr := scheduler_scheduleonetime(delaySeconds, payloadMem.Offset(), scheduleIDMem.Offset())
	responseMem := pdk.FindMemory(responsePtr)
	responseBytes := responseMem.ReadBytes()

	var resp SchedulerScheduleOneTimeResponse
	if err := json.Unmarshal(responseBytes, &resp); err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.NewScheduleID, nil
}

// SchedulerScheduleRecurring schedules a recurring task using a cron expression.
func SchedulerScheduleRecurring(cronExpression, payload, scheduleID string) (string, error) {
	cronMem := pdk.AllocateString(cronExpression)
	defer cronMem.Free()
	payloadMem := pdk.AllocateString(payload)
	defer payloadMem.Free()
	scheduleIDMem := pdk.AllocateString(scheduleID)
	defer scheduleIDMem.Free()

	responsePtr := scheduler_schedulerecurring(cronMem.Offset(), payloadMem.Offset(), scheduleIDMem.Offset())
	responseMem := pdk.FindMemory(responsePtr)
	responseBytes := responseMem.ReadBytes()

	var resp SchedulerScheduleRecurringResponse
	if err := json.Unmarshal(responseBytes, &resp); err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.NewScheduleID, nil
}

// SchedulerCancelSchedule cancels a scheduled task.
func SchedulerCancelSchedule(scheduleID string) error {
	scheduleIDMem := pdk.AllocateString(scheduleID)
	defer scheduleIDMem.Free()

	responsePtr := scheduler_cancelschedule(scheduleIDMem.Offset())
	if responsePtr != 0 {
		responseMem := pdk.FindMemory(responsePtr)
		errStr := string(responseMem.ReadBytes())
		if errStr != "" {
			return errors.New(errStr)
		}
	}
	return nil
}
