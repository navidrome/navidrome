package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerateSchema(t *testing.T) {
	tests := []struct {
		name       string
		capability Capability
		wantErr    bool
		validate   func(t *testing.T, schema []byte)
	}{
		{
			name: "basic capability with one export",
			capability: Capability{
				Name:       "test",
				Doc:        "Test capability",
				SourceFile: "test",
				Methods: []Export{
					{
						ExportName: "test_method",
						Doc:        "Test method does something",
						Input:      NewParam("input", "TestInput"),
						Output:     NewParam("output", "TestOutput"),
					},
				},
				Structs: []StructDef{
					{
						Name: "TestInput",
						Doc:  "Input for test",
						Fields: []FieldDef{
							{Name: "Name", Type: "string", JSONTag: "name", Doc: "The name"},
							{Name: "Count", Type: "int", JSONTag: "count", Doc: "The count"},
						},
					},
					{
						Name: "TestOutput",
						Doc:  "Output for test",
						Fields: []FieldDef{
							{Name: "Result", Type: "string", JSONTag: "result", Doc: "The result"},
						},
					},
				},
			},
			validate: func(t *testing.T, schema []byte) {
				var doc map[string]any
				require.NoError(t, yaml.Unmarshal(schema, &doc))

				// Check version
				assert.Equal(t, "v1-draft", doc["version"])

				// Check exports
				exports := doc["exports"].(map[string]any)
				assert.Contains(t, exports, "test_method")
				method := exports["test_method"].(map[string]any)
				assert.Equal(t, "Test method does something", method["description"])

				// Check schemas
				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				assert.Contains(t, schemas, "TestInput")
				assert.Contains(t, schemas, "TestOutput")

				// Check TestInput schema
				input := schemas["TestInput"].(map[string]any)
				assert.Equal(t, "object", input["type"]) // Workaround for XTP code generator
				props := input["properties"].(map[string]any)
				assert.Contains(t, props, "name")
				assert.Contains(t, props, "count")

				// Check required fields (non-pointer, non-omitempty)
				required := input["required"].([]any)
				assert.Contains(t, required, "name")
				assert.Contains(t, required, "count")
			},
		},
		{
			name: "capability with pointer fields (nullable)",
			capability: Capability{
				Name:       "nullable_test",
				SourceFile: "nullable_test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
				},
				Structs: []StructDef{
					{
						Name: "Input",
						Fields: []FieldDef{
							{Name: "Required", Type: "string", JSONTag: "required"},
							{Name: "Optional", Type: "*string", JSONTag: "optional,omitempty", OmitEmpty: true},
						},
					},
					{
						Name: "Output",
						Fields: []FieldDef{
							{Name: "Value", Type: "string", JSONTag: "value"},
						},
					},
				},
			},
			validate: func(t *testing.T, schema []byte) {
				var doc map[string]any
				require.NoError(t, yaml.Unmarshal(schema, &doc))

				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)

				// Required field is not nullable
				requiredField := props["required"].(map[string]any)
				assert.NotContains(t, requiredField, "nullable")

				// Optional pointer field is nullable
				optionalField := props["optional"].(map[string]any)
				assert.Equal(t, true, optionalField["nullable"])

				// Check required array only has non-pointer fields
				required := input["required"].([]any)
				assert.Contains(t, required, "required")
				assert.NotContains(t, required, "optional")
			},
		},
		{
			name: "capability with enum",
			capability: Capability{
				Name:       "enum_test",
				SourceFile: "enum_test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
				},
				Structs: []StructDef{
					{
						Name: "Input",
						Fields: []FieldDef{
							{Name: "Status", Type: "Status", JSONTag: "status"},
						},
					},
					{
						Name: "Output",
						Fields: []FieldDef{
							{Name: "Value", Type: "string", JSONTag: "value"},
						},
					},
				},
				TypeAliases: []TypeAlias{
					{Name: "Status", Type: "string", Doc: "Status type"},
				},
				Consts: []ConstGroup{
					{
						Type: "Status",
						Values: []ConstDef{
							{Name: "StatusPending", Value: `"pending"`},
							{Name: "StatusActive", Value: `"active"`},
							{Name: "StatusDone", Value: `"done"`},
						},
					},
				},
			},
			validate: func(t *testing.T, schema []byte) {
				var doc map[string]any
				require.NoError(t, yaml.Unmarshal(schema, &doc))

				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)

				// Check enum is defined
				assert.Contains(t, schemas, "Status")
				status := schemas["Status"].(map[string]any)
				assert.Equal(t, "string", status["type"])
				enum := status["enum"].([]any)
				assert.ElementsMatch(t, []any{"pending", "active", "done"}, enum)

				// Check $ref in Input
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)
				statusRef := props["status"].(map[string]any)
				assert.Equal(t, "#/components/schemas/Status", statusRef["$ref"])
			},
		},
		{
			name: "capability with array types",
			capability: Capability{
				Name:       "array_test",
				SourceFile: "array_test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
				},
				Structs: []StructDef{
					{
						Name: "Input",
						Fields: []FieldDef{
							{Name: "Tags", Type: "[]string", JSONTag: "tags"},
							{Name: "Items", Type: "[]Item", JSONTag: "items"},
						},
					},
					{
						Name: "Output",
						Fields: []FieldDef{
							{Name: "Value", Type: "string", JSONTag: "value"},
						},
					},
					{
						Name: "Item",
						Fields: []FieldDef{
							{Name: "ID", Type: "string", JSONTag: "id"},
						},
					},
				},
			},
			validate: func(t *testing.T, schema []byte) {
				var doc map[string]any
				require.NoError(t, yaml.Unmarshal(schema, &doc))

				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				input := schemas["Input"].(map[string]any)
				props := input["properties"].(map[string]any)

				// Check string array
				tags := props["tags"].(map[string]any)
				assert.Equal(t, "array", tags["type"])
				tagItems := tags["items"].(map[string]any)
				assert.Equal(t, "string", tagItems["type"])

				// Check struct array (uses $ref)
				items := props["items"].(map[string]any)
				assert.Equal(t, "array", items["type"])
				itemItems := items["items"].(map[string]any)
				assert.Equal(t, "#/components/schemas/Item", itemItems["$ref"])
			},
		},
		{
			name: "capability with nullable ref",
			capability: Capability{
				Name:       "nullable_ref_test",
				SourceFile: "nullable_ref_test",
				Methods: []Export{
					{ExportName: "test", Input: NewParam("input", "Input"), Output: NewParam("output", "Output")},
				},
				Structs: []StructDef{
					{
						Name: "Input",
						Fields: []FieldDef{
							{Name: "Value", Type: "string", JSONTag: "value"},
						},
					},
					{
						Name: "Output",
						Fields: []FieldDef{
							{Name: "Status", Type: "*ErrorType", JSONTag: "status,omitempty", OmitEmpty: true},
						},
					},
				},
				TypeAliases: []TypeAlias{
					{Name: "ErrorType", Type: "string"},
				},
				Consts: []ConstGroup{
					{
						Type: "ErrorType",
						Values: []ConstDef{
							{Name: "ErrorNone", Value: `"none"`},
							{Name: "ErrorFatal", Value: `"fatal"`},
						},
					},
				},
			},
			validate: func(t *testing.T, schema []byte) {
				var doc map[string]any
				require.NoError(t, yaml.Unmarshal(schema, &doc))

				components := doc["components"].(map[string]any)
				schemas := components["schemas"].(map[string]any)
				output := schemas["Output"].(map[string]any)
				props := output["properties"].(map[string]any)

				// Pointer to enum type should have $ref AND nullable
				status := props["status"].(map[string]any)
				assert.Equal(t, "#/components/schemas/ErrorType", status["$ref"])
				assert.Equal(t, true, status["nullable"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := GenerateSchema(tt.capability)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, schema)

			if tt.validate != nil {
				tt.validate(t, schema)
			}
		})
	}
}

func TestGoTypeToXTPTypeAndFormat(t *testing.T) {
	tests := []struct {
		goType     string
		wantType   string
		wantFormat string
	}{
		{"string", "string", ""},
		{"int", "integer", "int32"},
		{"int32", "integer", "int32"},
		{"int64", "integer", "int64"},
		{"float32", "number", "float"},
		{"float64", "number", "float"},
		{"bool", "boolean", ""},
		{"[]byte", "string", "byte"},
		// Unknown types default to object
		{"CustomType", "object", ""},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			gotType, gotFormat := goTypeToXTPTypeAndFormat(tt.goType)
			assert.Equal(t, tt.wantType, gotType)
			assert.Equal(t, tt.wantFormat, gotFormat)
		})
	}
}

func TestCleanDocForYAML(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want string
	}{
		{
			name: "empty",
			doc:  "",
			want: "",
		},
		{
			name: "single line",
			doc:  "Simple description",
			want: "Simple description",
		},
		{
			name: "multiline",
			doc:  "First line\nSecond line",
			want: "First line\nSecond line",
		},
		{
			name: "trailing newline",
			doc:  "Description\n",
			want: "Description",
		},
		{
			name: "whitespace",
			doc:  "  Description  ",
			want: "Description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanDocForYAML(tt.doc)
			assert.Equal(t, tt.want, got)
		})
	}
}
