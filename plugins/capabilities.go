package plugins

import (
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/core/agents"
)

// Capability represents a plugin capability type.
// Capabilities are detected by checking which functions a plugin exports.
type Capability string

// capabilityFunctions maps each capability to its required/optional functions.
// A plugin has a capability if it exports at least one of these functions.
var capabilityFunctions = map[Capability][]string{}

// registerCapability registers a capability with its associated functions.
func registerCapability(cap Capability, functions ...string) {
	capabilityFunctions[cap] = functions
}

// functionExistsChecker is an interface for checking if a function exists in a plugin.
// This allows for testing without a real plugin instance.
type functionExistsChecker interface {
	FunctionExists(name string) bool
}

// detectCapabilities detects which capabilities a plugin has by checking
// which functions it exports.
func detectCapabilities(plugin functionExistsChecker) []Capability {
	var capabilities []Capability

	for cap, functions := range capabilityFunctions {
		if slices.ContainsFunc(functions, plugin.FunctionExists) {
			capabilities = append(capabilities, cap) // Found at least one function, plugin has this capability
		}
	}

	return capabilities
}

// hasCapability checks if the given capabilities slice contains a specific capability.
func hasCapability(capabilities []Capability, cap Capability) bool {
	return slices.Contains(capabilities, cap)
}

// retryLaterRe matches the `<capability>(retry_later[:seconds])` token a plugin can put in
// its error string, which is all a plugin fault carries back across the WASM boundary.
var retryLaterRe = regexp.MustCompile(`(\w+)\(retry_later(?::(\d+))?\)`)

// parseRetryLater reports whether msg carries prefix's retry_later token, with its delay.
func parseRetryLater(prefix, msg string) (*agents.RetryLaterError, bool) {
	m := retryLaterRe.FindStringSubmatch(msg)
	if m == nil || m[1] != prefix {
		return nil, false
	}
	retry := &agents.RetryLaterError{}
	if secs, err := strconv.Atoi(m[2]); err == nil && secs > 0 {
		retry.RetryIn = time.Duration(min(secs, agents.MaxRetryInSeconds)) * time.Second
	}
	return retry, true
}
