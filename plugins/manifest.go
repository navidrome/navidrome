package plugins

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:generate go tool go-jsonschema -p plugins --struct-name-from-title -o manifest_gen.go manifest-schema.json

// ParseManifest unmarshals manifest JSON and performs cross-field validation.
// This is the single entry point for manifest parsing after reading from a file.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest JSON: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}
	return &m, nil
}

// Validate performs cross-field validation that cannot be expressed in JSON Schema.
// This validates rules like "SubsonicAPI permission requires users permission".
func (m *Manifest) Validate() error {
	// SubsonicAPI permission requires users permission
	if m.Permissions != nil && m.Permissions.Subsonicapi != nil {
		if m.Permissions.Users == nil {
			return fmt.Errorf("'subsonicapi' permission requires 'users' permission to be declared")
		}
	}

	// Validate config schema if present
	if m.Config != nil && m.Config.Schema != nil {
		if err := validateConfigSchema(m.Config.Schema); err != nil {
			return fmt.Errorf("invalid config schema: %w", err)
		}
	}

	return nil
}

// validateConfigSchema validates that the schema is a valid JSON Schema that can be compiled.
func validateConfigSchema(schema map[string]any) error {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", schema); err != nil {
		return fmt.Errorf("invalid schema structure: %w", err)
	}
	if _, err := compiler.Compile("schema.json"); err != nil {
		return err
	}
	return nil
}

// ValidateWithCapabilities validates the manifest against detected capabilities.
// This must be called after WASM capability detection since Scrobbler capability
// is detected from exported functions, not manifest declarations.
func ValidateWithCapabilities(m *Manifest, capabilities []Capability) error {
	// Scrobbler capability requires users permission
	if hasCapability(capabilities, CapabilityScrobbler) {
		if m.Permissions == nil || m.Permissions.Users == nil {
			return fmt.Errorf("scrobbler capability requires 'users' permission to be declared in manifest")
		}
	}
	return nil
}

// HasExperimentalThreads returns true if the manifest requests experimental threads support.
func (m *Manifest) HasExperimentalThreads() bool {
	return m.Experimental != nil && m.Experimental.Threads != nil
}
