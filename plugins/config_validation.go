package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ConfigValidationError represents a validation error with field path and message.
type ConfigValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ConfigValidationErrors is a collection of validation errors.
type ConfigValidationErrors struct {
	Errors []ConfigValidationError `json:"errors"`
}

func (e *ConfigValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	var msgs []string
	for _, err := range e.Errors {
		if err.Field != "" {
			msgs = append(msgs, fmt.Sprintf("%s: %s", err.Field, err.Message))
		} else {
			msgs = append(msgs, err.Message)
		}
	}
	return strings.Join(msgs, "; ")
}

// ValidateConfig validates a config JSON string against a plugin's config schema.
// If the manifest has no config schema, it returns an error indicating the plugin
// has no configurable options.
// Returns nil if validation passes, ConfigValidationErrors if validation fails.
func ValidateConfig(manifest *Manifest, configJSON string) error {
	// If no config schema defined, plugin has no configurable options
	if !manifest.HasConfigSchema() {
		return fmt.Errorf("plugin has no configurable options")
	}

	// Parse the config JSON (empty string treated as empty object)
	var configData any
	if configJSON == "" {
		configData = map[string]any{}
	} else {
		if err := json.Unmarshal([]byte(configJSON), &configData); err != nil {
			return &ConfigValidationErrors{
				Errors: []ConfigValidationError{{
					Message: fmt.Sprintf("invalid JSON: %v", err),
				}},
			}
		}
	}

	// Compile the schema
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", manifest.Config.Schema); err != nil {
		return fmt.Errorf("adding schema resource: %w", err)
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("compiling schema: %w", err)
	}

	// Validate config against schema
	if err := schema.Validate(configData); err != nil {
		return convertValidationError(err)
	}

	return nil
}

// convertValidationError converts jsonschema validation errors to our format.
func convertValidationError(err error) *ConfigValidationErrors {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return &ConfigValidationErrors{
			Errors: []ConfigValidationError{{
				Message: err.Error(),
			}},
		}
	}

	var configErrors []ConfigValidationError
	collectErrors(validationErr, &configErrors)

	if len(configErrors) == 0 {
		configErrors = append(configErrors, ConfigValidationError{
			Message: validationErr.Error(),
		})
	}

	return &ConfigValidationErrors{Errors: configErrors}
}

// collectErrors recursively collects validation errors from the error tree.
func collectErrors(err *jsonschema.ValidationError, errors *[]ConfigValidationError) {
	// If there are child errors, collect from them
	if len(err.Causes) > 0 {
		for _, cause := range err.Causes {
			collectErrors(cause, errors)
		}
		return
	}

	// Leaf error - add it
	field := ""
	if len(err.InstanceLocation) > 0 {
		field = strings.Join(err.InstanceLocation, "/")
	}

	*errors = append(*errors, ConfigValidationError{
		Field:   field,
		Message: err.Error(),
	})
}

// HasConfigSchema returns true if the manifest defines a config schema.
func (m *Manifest) HasConfigSchema() bool {
	return m.Config != nil && m.Config.Schema != nil
}
