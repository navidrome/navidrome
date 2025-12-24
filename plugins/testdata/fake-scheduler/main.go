// Fake scheduler plugin for Navidrome plugin system integration tests.
// This plugin was created based on the scheduler_callback.yaml XTP schema.
// Build with: tinygo build -o ../fake-scheduler.wasm -target wasip1 -buildmode=c-shared .
//
// Note: pdk.gen.go contains the domain types from the XTP schema where your plugin will run.
package main

import (
	"encoding/json"
	"strconv"

	pdk "github.com/extism/go-pdk"
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

// CallRecord stores information about a callback that was received (for testing)
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

// NdSchedulerCallback implements the scheduler callback logic.
// Called when a scheduled task fires.
// This function is called by the generated wrapper in pdk.gen.go.
func NdSchedulerCallback(input SchedulerCallbackInput) (SchedulerCallbackOutput, error) {
	// Check for configured error response
	errCfg, hasErr := pdk.GetConfig("callback_error")
	if hasErr && errCfg != "" {
		return SchedulerCallbackOutput{Error: &errCfg}, nil
	}

	// Track the callback
	totalCallCount++
	if record, exists := callRecords[input.ScheduleId]; exists {
		record.CallCount++
	} else {
		callRecords[input.ScheduleId] = &CallRecord{
			ScheduleID:  input.ScheduleId,
			Payload:     input.Payload,
			IsRecurring: input.IsRecurring,
			CallCount:   1,
		}
	}

	// Log the callback for debugging
	pdk.Log(pdk.LogInfo, "Scheduler callback received: "+input.ScheduleId+" payload="+input.Payload)

	return SchedulerCallbackOutput{}, nil
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
