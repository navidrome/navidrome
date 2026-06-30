package plugins

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// DeclaredNames returns the sorted names of the non-nil permission fields. It
// reflects over the generated json tags so new permission types are picked up
// automatically rather than via a hand-maintained list.
func (p *Permissions) DeclaredNames() []string {
	if p == nil {
		return nil
	}
	var names []string
	v := reflect.ValueOf(*p)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() != reflect.Pointer || f.IsNil() {
			continue
		}
		tag := t.Field(i).Tag.Get("json")
		if name, _, _ := strings.Cut(tag, ","); name != "" && name != "-" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

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

	// Matcher returns library content, so it requires the library permission (which
	// is what exposes a library scope for configuration).
	if m.Permissions != nil && m.Permissions.Matcher != nil {
		if m.Permissions.Library == nil {
			return fmt.Errorf("'matcher' permission requires 'library' permission to be declared")
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

	// Scheduler permission requires SchedulerCallback capability
	if m.Permissions != nil && m.Permissions.Scheduler != nil {
		if !hasCapability(capabilities, CapabilityScheduler) {
			return fmt.Errorf("'scheduler' permission requires plugin to export '%s' function", FuncSchedulerCallback)
		}
	}

	// Task (taskqueue) permission requires TaskWorker capability
	if m.Permissions != nil && m.Permissions.Taskqueue != nil {
		if !hasCapability(capabilities, CapabilityTaskWorker) {
			return fmt.Errorf("'taskqueue' permission requires plugin to export '%s' function", FuncTaskWorkerCallback)
		}
	}

	return nil
}

// HasExperimentalThreads returns true if the manifest requests experimental threads support.
func (m *Manifest) HasExperimentalThreads() bool {
	return m.Experimental != nil && m.Experimental.Threads != nil
}

// HasLibraryFilesystemPermission checks if the manifest grants filesystem permission for libraries.
func (m *Manifest) HasLibraryFilesystemPermission() bool {
	return m.Permissions != nil &&
		m.Permissions.Library != nil &&
		m.Permissions.Library.Filesystem
}
