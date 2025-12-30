package internal

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// XTP Schema types for YAML marshalling
type (
	xtpSchema struct {
		Version    string         `yaml:"version"`
		Exports    yaml.Node      `yaml:"exports,omitempty"`
		Components *xtpComponents `yaml:"components,omitempty"`
	}

	xtpComponents struct {
		Schemas yaml.Node `yaml:"schemas"`
	}

	xtpExport struct {
		Description string      `yaml:"description,omitempty"`
		Input       *xtpIOParam `yaml:"input,omitempty"`
		Output      *xtpIOParam `yaml:"output,omitempty"`
	}

	xtpIOParam struct {
		Ref         string `yaml:"$ref"`
		ContentType string `yaml:"contentType"`
	}

	// xtpObjectSchema represents an object schema in XTP.
	// Note: The Type field is technically not valid per the XTP JSON Schema,
	// but is required as a workaround for XTP's code generator to properly
	// resolve type information (especially for empty structs).
	xtpObjectSchema struct {
		Description string    `yaml:"description,omitempty"`
		Type        string    `yaml:"type"`
		Properties  yaml.Node `yaml:"properties"`
		Required    []string  `yaml:"required,omitempty"`
	}

	xtpEnumSchema struct {
		Description string   `yaml:"description,omitempty"`
		Type        string   `yaml:"type"`
		Enum        []string `yaml:"enum"`
	}

	xtpProperty struct {
		Ref         string       `yaml:"$ref,omitempty"`
		Type        string       `yaml:"type,omitempty"`
		Format      string       `yaml:"format,omitempty"`
		Description string       `yaml:"description,omitempty"`
		Nullable    bool         `yaml:"nullable,omitempty"`
		Items       *xtpProperty `yaml:"items,omitempty"`
	}
)

// GenerateSchema generates an XTP YAML schema from a capability.
func GenerateSchema(cap Capability) ([]byte, error) {
	schema := xtpSchema{Version: "v1-draft"}

	// Build exports as ordered map
	if len(cap.Methods) > 0 {
		schema.Exports = yaml.Node{Kind: yaml.MappingNode}
		for _, export := range cap.Methods {
			addToMap(&schema.Exports, export.ExportName, buildExport(export))
		}
	}

	// Build components/schemas
	schemas := buildSchemas(cap)
	if len(schemas.Content) > 0 {
		schema.Components = &xtpComponents{Schemas: schemas}
	}

	return yaml.Marshal(schema)
}

func buildExport(export Export) xtpExport {
	e := xtpExport{Description: cleanDocForYAML(export.Doc)}
	if export.Input.Type != "" {
		e.Input = &xtpIOParam{
			Ref:         "#/components/schemas/" + strings.TrimPrefix(export.Input.Type, "*"),
			ContentType: "application/json",
		}
	}
	if export.Output.Type != "" {
		e.Output = &xtpIOParam{
			Ref:         "#/components/schemas/" + strings.TrimPrefix(export.Output.Type, "*"),
			ContentType: "application/json",
		}
	}
	return e
}

func buildSchemas(cap Capability) yaml.Node {
	schemas := yaml.Node{Kind: yaml.MappingNode}
	knownTypes := cap.KnownStructs()
	for _, alias := range cap.TypeAliases {
		knownTypes[alias.Name] = true
	}

	// Sort structs by name for consistent output
	structNames := make([]string, 0, len(cap.Structs))
	structMap := make(map[string]StructDef)
	for _, st := range cap.Structs {
		structNames = append(structNames, st.Name)
		structMap[st.Name] = st
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		st := structMap[name]
		addToMap(&schemas, name, buildObjectSchema(st, knownTypes))
	}

	// Build enum types from type aliases
	for _, alias := range cap.TypeAliases {
		if alias.Type == "string" {
			for _, cg := range cap.Consts {
				if cg.Type == alias.Name {
					addToMap(&schemas, alias.Name, buildEnumSchema(alias, cg))
					break
				}
			}
		}
	}

	return schemas
}

func buildObjectSchema(st StructDef, knownTypes map[string]bool) xtpObjectSchema {
	schema := xtpObjectSchema{
		Description: cleanDocForYAML(st.Doc),
		Type:        "object", // Required workaround for XTP code generator
		Properties:  yaml.Node{Kind: yaml.MappingNode},
	}

	for _, field := range st.Fields {
		propName := getJSONFieldName(field)
		addToMap(&schema.Properties, propName, buildProperty(field, knownTypes))

		if !strings.HasPrefix(field.Type, "*") && !field.OmitEmpty {
			schema.Required = append(schema.Required, propName)
		}
	}

	return schema
}

func buildEnumSchema(alias TypeAlias, cg ConstGroup) xtpEnumSchema {
	values := make([]string, 0, len(cg.Values))
	for _, cv := range cg.Values {
		values = append(values, strings.Trim(cv.Value, `"`))
	}
	return xtpEnumSchema{
		Description: cleanDocForYAML(alias.Doc),
		Type:        "string",
		Enum:        values,
	}
}

func buildProperty(field FieldDef, knownTypes map[string]bool) xtpProperty {
	goType := field.Type
	isPointer := strings.HasPrefix(goType, "*")
	if isPointer {
		goType = goType[1:]
	}

	prop := xtpProperty{
		Description: cleanDocForYAML(field.Doc),
		Nullable:    isPointer,
	}

	// Handle reference types (use $ref instead of type)
	if isKnownType(goType, knownTypes) && !strings.HasPrefix(goType, "[]") {
		prop.Ref = "#/components/schemas/" + goType
		return prop
	}

	// Handle slice types
	if strings.HasPrefix(goType, "[]") {
		elemType := goType[2:]
		prop.Type = "array"
		prop.Items = &xtpProperty{}
		if isKnownType(elemType, knownTypes) {
			prop.Items.Ref = "#/components/schemas/" + elemType
		} else {
			prop.Items.Type = goTypeToXTPType(elemType)
		}
		return prop
	}

	// Handle primitive types
	prop.Type, prop.Format = goTypeToXTPTypeAndFormat(goType)
	return prop
}

// addToMap adds a key-value pair to a yaml.Node map, preserving insertion order.
func addToMap[T any](node *yaml.Node, key string, value T) {
	var valNode yaml.Node
	_ = valNode.Encode(value)
	node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, &valNode)
}

func getJSONFieldName(field FieldDef) string {
	propName := field.JSONTag
	if idx := strings.Index(propName, ","); idx >= 0 {
		propName = propName[:idx]
	}
	if propName == "" {
		propName = field.Name
	}
	return propName
}

// isKnownType checks if a type is a known struct or type alias.
func isKnownType(typeName string, knownTypes map[string]bool) bool {
	return knownTypes[typeName]
}

// goTypeToXTPType converts a Go type to an XTP schema type.
func goTypeToXTPType(goType string) string {
	typ, _ := goTypeToXTPTypeAndFormat(goType)
	return typ
}

// goTypeToXTPTypeAndFormat converts a Go type to XTP type and format.
func goTypeToXTPTypeAndFormat(goType string) (typ, format string) {
	switch goType {
	case "string":
		return "string", ""
	case "int", "int32":
		return "integer", "int32"
	case "int64":
		return "integer", "int64"
	case "float32":
		return "number", "float"
	case "float64":
		return "number", "float"
	case "bool":
		return "boolean", ""
	case "[]byte":
		return "string", "byte"
	default:
		return "object", ""
	}
}

// cleanDocForYAML cleans documentation for YAML output.
func cleanDocForYAML(doc string) string {
	doc = strings.TrimSpace(doc)
	// Remove leading "// " from each line if present
	lines := strings.Split(doc, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(strings.TrimSpace(line), "// ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
