// Test scheduler plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-scheduler.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
)

func init() {
	scheduler.Register(&testScheduler{})
}

type testScheduler struct{}

// OnCallback is called when a scheduled task fires.
// Magic payloads trigger specific behaviors to test host functions:
//   - "schedule-followup": schedules a one-time task via host function
//   - "schedule-recurring": schedules a recurring task via host function
//   - "schedule-duplicate:<id>": attempts to schedule with the given ID (for testing duplicate detection)
func (t *testScheduler) OnCallback(input scheduler.SchedulerCallbackRequest) error {
	switch {
	case input.Payload == "schedule-followup":
		if _, err := host.SchedulerScheduleOneTime(1, "followup-created", "followup-id"); err != nil {
			return err
		}
	case input.Payload == "schedule-recurring":
		if _, err := host.SchedulerScheduleRecurring("@every 1s", "recurring-created", "recurring-from-plugin"); err != nil {
			return err
		}
	case len(input.Payload) > 19 && input.Payload[:19] == "schedule-duplicate:":
		duplicateID := input.Payload[19:]
		if _, err := host.SchedulerScheduleOneTime(60, "duplicate-attempt", duplicateID); err != nil {
			return err
		}
	}
	return nil
}

func main() {}
