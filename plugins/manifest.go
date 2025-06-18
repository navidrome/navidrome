package plugins

//go:generate go tool go-jsonschema --schema-root-type navidrome://plugins/manifest=PluginManifest -p schema --output schema/manifest_gen.go schema/manifest.schema.json

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/plugins/schema"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schema/manifest.schema.json
var schemaData []byte

type PluginManifest struct {
	Name         string                           `json:"name"`
	Author       string                           `json:"author"`
	Version      string                           `json:"version"`
	Description  string                           `json:"description"`
	Capabilities []string                         `json:"capabilities"`
	Permissions  schema.PluginManifestPermissions `json:"permissions"`
}

// LoadManifest loads and parses the manifest.json file from the given plugin directory.
func LoadManifest(pluginDir string) (*PluginManifest, error) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	// Validate against schema
	if err := validateManifest(data); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &manifest, nil
}

// validateManifest validates the manifest JSON against the schema
func validateManifest(data []byte) error {
	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	documentLoader := gojsonschema.NewBytesLoader(data)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("error validating manifest: %w", err)
	}

	if !result.Valid() {
		var errorDetails []string
		for _, err := range result.Errors() {
			// Format detailed error message
			field := err.Field()
			message := err.Description()
			if err.Value() != nil {
				message = fmt.Sprintf("%s (got: %v)", message, err.Value())
			}
			errorDetails = append(errorDetails, fmt.Sprintf("%s: %s", field, message))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(errorDetails, "; "))
	}

	return nil
}
