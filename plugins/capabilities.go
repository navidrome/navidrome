package plugins

// Capability represents a plugin capability type.
// Capabilities are detected by checking which functions a plugin exports.
type Capability string

const (
	// CapabilityMetadataAgent indicates the plugin can provide artist/album metadata.
	// Detected when the plugin exports at least one of the metadata agent functions.
	CapabilityMetadataAgent Capability = "MetadataAgent"

	// CapabilityScrobbler indicates the plugin can receive scrobble events.
	// Detected when the plugin exports at least one of the scrobbler functions.
	CapabilityScrobbler Capability = "Scrobbler"

	// CapabilityScheduler indicates the plugin can receive scheduled event callbacks.
	// Detected when the plugin exports the scheduler callback function.
	CapabilityScheduler Capability = "Scheduler"
)

// capabilityFunctions maps each capability to its required/optional functions.
// A plugin has a capability if it exports at least one of these functions.
var capabilityFunctions = map[Capability][]string{
	CapabilityMetadataAgent: {
		FuncGetArtistMBID,
		FuncGetArtistURL,
		FuncGetArtistBiography,
		FuncGetSimilarArtists,
		FuncGetArtistImages,
		FuncGetArtistTopSongs,
		FuncGetAlbumInfo,
		FuncGetAlbumImages,
	},
	CapabilityScrobbler: {
		FuncScrobblerIsAuthorized,
		FuncScrobblerNowPlaying,
		FuncScrobblerScrobble,
	},
	CapabilityScheduler: {
		FuncSchedulerCallback,
	},
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
