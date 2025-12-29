// Test scheduler plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-scheduler.wasm -target wasip1 -buildmode=c-shared .
package main

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
