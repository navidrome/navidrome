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
		Ref         string `yaml:"$ref,omitempty"`
		Type        string `yaml:"type,omitempty"`
		ContentType string `yaml:"contentType"`
	}

	// xtpObjectSchema represents an object schema in XTP.
	// Per the XTP JSON Schema, ObjectSchema has properties, required, and description
	// but NOT a type field.
	xtpObjectSchema struct {
		Description string    `yaml:"description,omitempty"`
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

	aliasToCanonical := buildAliasToCanonical(cap)

	// Build exports as ordered map
	if len(cap.Methods) > 0 {
		schema.Exports = yaml.Node{Kind: yaml.MappingNode}
		for _, export := range cap.Methods {
			addToMap(&schema.Exports, export.ExportName, buildExport(export, aliasToCanonical))
		}
	}

	// Build components/schemas
	schemas := buildSchemas(cap, aliasToCanonical)
	if len(schemas.Content) > 0 {
		schema.Components = &xtpComponents{Schemas: schemas}
	}

	return yaml.Marshal(schema)
}

// buildAliasToCanonical maps each deprecated shared-alias name to the canonical
// shared type it targets (e.g. TrackInfo -> Track). Schema components are emitted
// under the canonical name, so every $ref site must resolve through this map.
func buildAliasToCanonical(cap Capability) map[string]string {
	m := map[string]string{}
	for _, a := range cap.SharedAliases {
		m[a.Name] = strings.TrimPrefix(a.Target, sharedTypesPrefix)
	}
	return m
}

func buildExport(export Export, aliasToCanonical map[string]string) xtpExport {
	e := xtpExport{Description: cleanDocForYAML(export.Doc)}
	if export.Input.Type != "" {
		e.Input = &xtpIOParam{
			Ref:         "#/components/schemas/" + canonicalRefName(fieldBaseType(export.Input.Type), aliasToCanonical),
			ContentType: "application/json",
		}
	}
	if export.Output.Type != "" {
		outputType := strings.TrimPrefix(export.Output.Type, "*")
		// Check if output is a primitive type
		if isPrimitiveGoType(outputType) {
			e.Output = &xtpIOParam{
				Type:        goTypeToXTPType(outputType),
				ContentType: "application/json",
			}
		} else {
			e.Output = &xtpIOParam{
				Ref:         "#/components/schemas/" + canonicalRefName(fieldBaseType(outputType), aliasToCanonical),
				ContentType: "application/json",
			}
		}
	}
	return e
}

// isPrimitiveGoType returns true if the Go type is a primitive type.
func isPrimitiveGoType(goType string) bool {
	switch goType {
	case "bool", "string", "int", "int32", "int64", "uint", "uint32", "uint64", "float32", "float64", "[]byte":
		return true
	}
	return false
}

func buildSchemas(cap Capability, aliasToCanonical map[string]string) yaml.Node {
	schemas := yaml.Node{Kind: yaml.MappingNode}
	knownTypes := cap.KnownStructs()
	for _, alias := range cap.TypeAliases {
		knownTypes[alias.Name] = true
	}

	// Register shared types under their canonical name (e.g. types.Track -> Track)
	// and stash their struct shapes for inlining. SharedTypes covers every used
	// shared type, including ones referenced directly as types.X with no declared
	// deprecated alias; SharedAliases is folded in for completeness.
	sharedDefs := map[string]StructDef{}
	for _, def := range cap.SharedTypes {
		knownTypes[def.Name] = true
		sharedDefs[def.Name] = def
	}
	for _, a := range cap.SharedAliases {
		canonical := strings.TrimPrefix(a.Target, sharedTypesPrefix)
		knownTypes[canonical] = true
		sharedDefs[canonical] = a.Def
	}

	// Collect types that are actually used by exports
	usedTypes := collectUsedTypes(cap, knownTypes, sharedDefs)

	// A used alias name (e.g. TrackInfo) implies its canonical component (Track) is
	// used, since the alias-typed field's $ref resolves to the canonical name.
	for alias, canonical := range aliasToCanonical {
		if usedTypes[alias] {
			usedTypes[canonical] = true
		}
	}

	// Sort structs by name for consistent output
	structNames := make([]string, 0, len(cap.Structs))
	structMap := make(map[string]StructDef)
	for _, st := range cap.Structs {
		if usedTypes[st.Name] {
			structNames = append(structNames, st.Name)
			structMap[st.Name] = st
		}
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		st := structMap[name]
		addToMap(&schemas, name, buildObjectSchema(st, knownTypes, aliasToCanonical))
	}

	// Emit components for used shared aliases (sorted for deterministic output).
	sharedNames := make([]string, 0, len(sharedDefs))
	for name, def := range sharedDefs {
		if usedTypes[name] && len(def.Fields) > 0 {
			sharedNames = append(sharedNames, name)
		}
	}
	sort.Strings(sharedNames)
	for _, name := range sharedNames {
		addToMap(&schemas, name, buildObjectSchema(sharedDefs[name], knownTypes, aliasToCanonical))
	}

	// Build enum types from type aliases (only if used by exports)
	for _, alias := range cap.TypeAliases {
		if !usedTypes[alias.Name] {
			continue
		}
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

// collectUsedTypes returns a set of type names that are reachable from exports.
func collectUsedTypes(cap Capability, knownTypes map[string]bool, sharedDefs map[string]StructDef) map[string]bool {
	used := make(map[string]bool)

	// Start with types directly referenced by exports
	for _, export := range cap.Methods {
		if export.Input.Type != "" {
			addTypeAndDeps(strings.TrimPrefix(export.Input.Type, "*"), cap, knownTypes, sharedDefs, used)
		}
		if export.Output.Type != "" {
			outputType := strings.TrimPrefix(export.Output.Type, "*")
			if !isPrimitiveGoType(outputType) {
				addTypeAndDeps(outputType, cap, knownTypes, sharedDefs, used)
			}
		}
	}

	return used
}

// addTypeAndDeps adds a type and all its dependencies to the used set.
func addTypeAndDeps(typeName string, cap Capability, knownTypes map[string]bool, sharedDefs map[string]StructDef, used map[string]bool) {
	typeName = strings.TrimPrefix(typeName, sharedTypesPrefix)
	if used[typeName] || !knownTypes[typeName] {
		return
	}
	used[typeName] = true

	// Walk fields of capability-local structs.
	for _, st := range cap.Structs {
		if st.Name == typeName {
			for _, field := range st.Fields {
				if base := fieldBaseType(field.Type); knownTypes[base] {
					addTypeAndDeps(base, cap, knownTypes, sharedDefs, used)
				}
			}
			return
		}
	}

	// Walk fields of shared structs so their nested refs are also marked used.
	if def, ok := sharedDefs[typeName]; ok {
		for _, field := range def.Fields {
			if base := fieldBaseType(field.Type); knownTypes[base] {
				addTypeAndDeps(base, cap, knownTypes, sharedDefs, used)
			}
		}
	}
}

// fieldBaseType reduces a field type to the base named type used for schema
// lookups: it strips a leading pointer/slice and any shared `types.` selector.
func fieldBaseType(goType string) string {
	goType = strings.TrimPrefix(goType, "*")
	goType = strings.TrimPrefix(goType, "[]")
	return strings.TrimPrefix(goType, sharedTypesPrefix)
}

// canonicalRefName resolves a deprecated shared-alias name to the canonical type
// the schema component is emitted under (e.g. TrackInfo -> Track). Non-alias
// names pass through unchanged, so $ref targets always point at an emitted
// component instead of a dangling alias name.
func canonicalRefName(name string, aliasToCanonical map[string]string) string {
	if canonical, ok := aliasToCanonical[name]; ok {
		return canonical
	}
	return name
}

func buildObjectSchema(st StructDef, knownTypes map[string]bool, aliasToCanonical map[string]string) xtpObjectSchema {
	schema := xtpObjectSchema{
		Description: cleanDocForYAML(st.Doc),
		Properties:  yaml.Node{Kind: yaml.MappingNode},
	}

	for _, field := range st.Fields {
		propName := getJSONFieldName(field)
		addToMap(&schema.Properties, propName, buildProperty(field, knownTypes, aliasToCanonical))

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

func buildProperty(field FieldDef, knownTypes map[string]bool, aliasToCanonical map[string]string) xtpProperty {
	goType := field.Type
	isPointer := strings.HasPrefix(goType, "*")
	if isPointer {
		goType = goType[1:]
	}

	prop := xtpProperty{
		Description: cleanDocForYAML(field.Doc),
		Nullable:    isPointer,
	}

	// Handle reference types (use $ref instead of type). Qualified shared
	// references (types.X) are referenced by their canonical name.
	if refType := strings.TrimPrefix(goType, sharedTypesPrefix); isKnownType(refType, knownTypes) && !strings.HasPrefix(goType, "[]") {
		prop.Ref = "#/components/schemas/" + canonicalRefName(refType, aliasToCanonical)
		return prop
	}

	// Handle primitive types (including []byte which maps to string/byte, not array)
	if isPrimitiveGoType(goType) {
		prop.Type, prop.Format = goTypeToXTPTypeAndFormat(goType)
		return prop
	}

	// Handle slice types
	if strings.HasPrefix(goType, "[]") {
		elemType := strings.TrimPrefix(goType[2:], sharedTypesPrefix)
		prop.Type = "array"
		prop.Items = &xtpProperty{}
		if isKnownType(elemType, knownTypes) {
			prop.Items.Ref = "#/components/schemas/" + canonicalRefName(elemType, aliasToCanonical)
		} else {
			prop.Items.Type = goTypeToXTPType(elemType)
		}
		return prop
	}

	// Handle remaining types
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
	case "uint", "uint32":
		// XTP schema doesn't support unsigned formats; use int64 to hold full uint32 range
		return "integer", "int64"
	case "uint64":
		// XTP schema doesn't support unsigned formats; use int64 (may lose precision for large values)
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
