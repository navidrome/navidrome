package plugins

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
		for _, fn := range functions {
			if plugin.FunctionExists(fn) {
				capabilities = append(capabilities, cap)
				break // Found at least one function, plugin has this capability
			}
		}
	}

	return capabilities
}

// hasCapability checks if the given capabilities slice contains a specific capability.
func hasCapability(capabilities []Capability, cap Capability) bool {
	for _, c := range capabilities {
		if c == cap {
			return true
		}
	}
	return false
}
