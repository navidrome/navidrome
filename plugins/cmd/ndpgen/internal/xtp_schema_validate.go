package internal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// XTP JSONSchema specification, from
// https://raw.githubusercontent.com/dylibso/xtp-bindgen/5090518dd86ba5e734dc225a33066ecc0ed2e12d/plugin/schema.json
//
//go:embed xtp_schema.json
var xtpSchemaJSON string

// ValidateXTPSchema validates that the generated schema conforms to the XTP JSONSchema specification.
// Returns nil if valid, or an error with validation details if invalid.
func ValidateXTPSchema(generatedSchema []byte) error {
	// Parse the YAML schema to JSON for validation
	var schemaDoc map[string]any
	if err := yaml.Unmarshal(generatedSchema, &schemaDoc); err != nil {
		return fmt.Errorf("failed to parse generated schema as YAML: %w", err)
	}

	// Convert to JSON for the validator
	jsonBytes, err := json.Marshal(schemaDoc)
	if err != nil {
		return fmt.Errorf("failed to convert schema to JSON: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(xtpSchemaJSON)
	documentLoader := gojsonschema.NewBytesLoader(jsonBytes)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	if !result.Valid() {
		var errs []string
		for _, desc := range result.Errors() {
			errs = append(errs, fmt.Sprintf("- %s", desc))
		}
		return fmt.Errorf("schema validation errors:\n%s", strings.Join(errs, "\n"))
	}

	return nil
}
