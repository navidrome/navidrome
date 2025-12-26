// Fake scheduler plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../fake-scheduler.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"encoding/json"

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

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:        "Fake Scheduler",
		Author:      "Navidrome Test",
		Version:     "1.0.0",
		Description: "A fake scheduler plugin for integration testing",
		Permissions: &Permissions{
			Scheduler: &SchedulerPermission{
				Reason: "For testing scheduler callbacks",
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

// NdSchedulerCallback is called when a scheduled task fires.
// Magic payloads trigger specific behaviors to test host functions:
//   - "schedule-followup": schedules a one-time task via host function
//   - "schedule-recurring": schedules a recurring task via host function
//   - "schedule-duplicate:<id>": attempts to schedule with the given ID (for testing duplicate detection)
func NdSchedulerCallback(input SchedulerCallbackInput) (SchedulerCallbackOutput, error) {
	switch {
	case input.Payload == "schedule-followup":
		_, err := SchedulerScheduleOneTime(1, "followup-created", "followup-id")
		if err != nil {
			errStr := err.Error()
			return SchedulerCallbackOutput{Error: &errStr}, nil
		}
	case input.Payload == "schedule-recurring":
		_, err := SchedulerScheduleRecurring("@every 1s", "recurring-created", "recurring-from-plugin")
		if err != nil {
			errStr := err.Error()
			return SchedulerCallbackOutput{Error: &errStr}, nil
		}
	case len(input.Payload) > 19 && input.Payload[:19] == "schedule-duplicate:":
		duplicateID := input.Payload[19:]
		_, err := SchedulerScheduleOneTime(60, "duplicate-attempt", duplicateID)
		if err != nil {
			errStr := err.Error()
			return SchedulerCallbackOutput{Error: &errStr}, nil
		}
	}
	return SchedulerCallbackOutput{}, nil
}

func main() {}
