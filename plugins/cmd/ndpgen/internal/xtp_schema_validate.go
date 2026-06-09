package internal

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
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

	// Parse the XTP schema JSON
	var xtpSchema any
	if err := json.Unmarshal([]byte(xtpSchemaJSON), &xtpSchema); err != nil {
		return fmt.Errorf("failed to parse XTP schema: %w", err)
	}

	// Compile the XTP schema
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("xtp-schema.json", xtpSchema); err != nil {
		return fmt.Errorf("failed to add XTP schema resource: %w", err)
	}

	schema, err := compiler.Compile("xtp-schema.json")
	if err != nil {
		return fmt.Errorf("failed to compile XTP schema: %w", err)
	}

	// Validate the generated schema against XTP schema
	if err := schema.Validate(schemaDoc); err != nil {
		return fmt.Errorf("schema validation errors:\n%s", formatValidationErrors(err))
	}

	return nil
}

// formatValidationErrors formats jsonschema validation errors into readable strings.
func formatValidationErrors(err error) string {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return fmt.Sprintf("- %s", err.Error())
	}

	var errs []string
	collectValidationErrors(validationErr, &errs)

	if len(errs) == 0 {
		return fmt.Sprintf("- %s", validationErr.Error())
	}
	return strings.Join(errs, "\n")
}

// collectValidationErrors recursively collects leaf validation errors.
func collectValidationErrors(err *jsonschema.ValidationError, errs *[]string) {
	if len(err.Causes) > 0 {
		for _, cause := range err.Causes {
			collectValidationErrors(cause, errs)
		}
		return
	}

	// Leaf error - format with location if available
	msg := err.Error()
	if len(err.InstanceLocation) > 0 {
		location := strings.Join(err.InstanceLocation, "/")
		msg = fmt.Sprintf("%s: %s", location, msg)
	}
	*errs = append(*errs, fmt.Sprintf("- %s", msg))
}
