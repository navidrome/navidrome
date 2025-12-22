package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rjeczalik/notify"
)

var _ = Describe("Watcher", func() {
	Describe("determinePluginAction", func() {
		// These are fast unit tests for the pure routing logic.
		// No WASM compilation, no file I/O - runs in microseconds.

		DescribeTable("returns correct action for event type and loaded state",
			func(eventType notify.Event, isLoaded bool, expected pluginAction) {
				Expect(determinePluginAction(eventType, isLoaded)).To(Equal(expected))
			},
			// CREATE events - always load
			Entry("CREATE when not loaded", notify.Create, false, actionLoad),
			Entry("CREATE when loaded", notify.Create, true, actionLoad),

			// WRITE events - reload if loaded, load if not
			Entry("WRITE when not loaded", notify.Write, false, actionLoad),
			Entry("WRITE when loaded", notify.Write, true, actionReload),

			// REMOVE events - always unload
			Entry("REMOVE when not loaded", notify.Remove, false, actionUnload),
			Entry("REMOVE when loaded", notify.Remove, true, actionUnload),

			// RENAME events - treated same as REMOVE
			Entry("RENAME when not loaded", notify.Rename, false, actionUnload),
			Entry("RENAME when loaded", notify.Rename, true, actionUnload),
		)

		It("returns actionNone for unknown event types", func() {
			// Event type 0 or other unknown values
			Expect(determinePluginAction(0, false)).To(Equal(actionNone))
			Expect(determinePluginAction(0, true)).To(Equal(actionNone))
		})
	})
})
