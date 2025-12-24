// Fake scheduler plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../fake-scheduler.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"encoding/json"
	"strconv"

	"github.com/extism/go-pdk"
)

// Manifest types
type Manifest struct {
	Name        string       `json:"name"`
	Author      string       `json:"author"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

type Permissions struct {
	Scheduler *SchedulerPermission `json:"scheduler,omitempty"`
}

type SchedulerPermission struct {
	Reason string `json:"reason,omitempty"`
}

// Scheduler callback input
type SchedulerCallbackInput struct {
	ScheduleID  string `json:"schedule_id"`
	Payload     string `json:"payload"`
	IsRecurring bool   `json:"is_recurring"`
}

// Scheduler callback output
type SchedulerCallbackOutput struct {
	Error string `json:"error,omitempty"`
}

// CallRecord stores information about a callback that was received
type CallRecord struct {
	ScheduleID  string `json:"schedule_id"`
	Payload     string `json:"payload"`
	IsRecurring bool   `json:"is_recurring"`
	CallCount   int    `json:"call_count"`
}

// Global state for tracking callbacks (var stores persist in wasm memory between calls)
var callRecords = make(map[string]*CallRecord)
var totalCallCount = 0

//go:wasmexport nd_manifest
func ndManifest() int32 {
	reason := "For testing scheduler callbacks"
	manifest := Manifest{
		Name:        "Fake Scheduler",
		Author:      "Navidrome Test",
		Version:     "1.0.0",
		Description: "A fake scheduler plugin for integration testing",
		Permissions: &Permissions{
			Scheduler: &SchedulerPermission{
				Reason: reason,
			},
		},
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	pdk.Output(out)
	return 0
}

//go:wasmexport nd_scheduler_callback
func ndSchedulerCallback() int32 {
	var input SchedulerCallbackInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	// Payload is now a plain string, no decoding needed
	payload := input.Payload

	// Check for configured error response
	errCfg, hasErr := pdk.GetConfig("callback_error")
	if hasErr && errCfg != "" {
		output := SchedulerCallbackOutput{Error: errCfg}
		if err := pdk.OutputJSON(output); err != nil {
			pdk.SetError(err)
			return 1
		}
		return 0
	}

	// Track the callback
	totalCallCount++
	if record, exists := callRecords[input.ScheduleID]; exists {
		record.CallCount++
	} else {
		callRecords[input.ScheduleID] = &CallRecord{
			ScheduleID:  input.ScheduleID,
			Payload:     payload,
			IsRecurring: input.IsRecurring,
			CallCount:   1,
		}
	}

	// Log the callback for debugging
	pdk.Log(pdk.LogInfo, "Scheduler callback received: "+input.ScheduleID+" payload="+payload)

	// Success
	output := SchedulerCallbackOutput{}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

// Helper function to get call records (for testing)
//
//go:wasmexport nd_get_call_records
func ndGetCallRecords() int32 {
	out, err := json.Marshal(callRecords)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	pdk.Output(out)
	return 0
}

// Helper function to get total call count (for testing)
//
//go:wasmexport nd_get_total_call_count
func ndGetTotalCallCount() int32 {
	pdk.Output([]byte(strconv.Itoa(totalCallCount)))
	return 0
}

// Helper function to reset call records (for testing)
//
//go:wasmexport nd_reset_call_records
func ndResetCallRecords() int32 {
	callRecords = make(map[string]*CallRecord)
	totalCallCount = 0
	return 0
}

func main() {}
