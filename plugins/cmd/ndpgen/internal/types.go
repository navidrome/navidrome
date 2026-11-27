package internal

import (
	"strings"
	"unicode"
)

// sharedTypesPrefix is the package selector a capability or host-service source
// uses to reference the shared types package (e.g. types.Track).
const sharedTypesPrefix = "types."

// Service represents a parsed host service interface.
type Service struct {
	Name          string        // Service name from annotation (e.g., "SubsonicAPI")
	Permission    string        // Manifest permission key (e.g., "subsonicapi")
	Interface     string        // Go interface name (e.g., "SubsonicAPIService")
	Methods       []Method      // Methods marked with //nd:hostfunc
	Doc           string        // Documentation comment for the service
	Structs       []StructDef   // Structs used by this service
	SharedAliases []SharedAlias // Aliases to types in the shared `types` package
}

// Capability represents a parsed capability interface for plugin exports.
type Capability struct {
	Name          string        // Package name from annotation (e.g., "metadata")
	Interface     string        // Go interface name (e.g., "MetadataAgent")
	Required      bool          // If true, all methods must be implemented
	Methods       []Export      // Methods marked with //nd:export
	Doc           string        // Documentation comment for the capability
	Structs       []StructDef   // Structs used by this capability
	TypeAliases   []TypeAlias   // Type aliases used by this capability
	Consts        []ConstGroup  // Const groups used by this capability
	SourceFile    string        // Base name of source file without extension (e.g., "websocket_callback")
	SharedAliases []SharedAlias // Aliases to types in the shared `types` package
	SharedTypes   []StructDef   // Resolved shapes of every used shared type, keyed by canonical name (for schema inlining, alias or not)
}

// TypeAlias represents a type declaration (e.g. type ScrobblerErrorType string)
// or a Go type alias (e.g. type TrackInfo = types.Track).
type TypeAlias struct {
	Name    string // Type name
	Type    string // Underlying type (or alias target, e.g. "types.Track")
	Doc     string // Documentation comment
	IsAlias bool   // true for `type X = Y` (alias); false for `type X Y` (defined type)
}

// IsDeprecated reports whether the alias carries a `Deprecated:` doc line.
func (t TypeAlias) IsDeprecated() bool {
	for _, line := range strings.Split(t.Doc, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Deprecated:") {
			return true
		}
	}
	return false
}

// IsSharedAlias reports whether this alias targets the shared types package
// (e.g. `type TrackInfo = types.Track`).
func (t TypeAlias) IsSharedAlias() bool {
	return t.IsAlias && strings.HasPrefix(t.Type, sharedTypesPrefix)
}

// ConstGroup represents a group of const definitions.
type ConstGroup struct {
	Type   string     // Type name for typed consts (empty for untyped)
	Values []ConstDef // Const definitions
}

// ConstDef represents a single const definition.
type ConstDef struct {
	Name  string // Const name
	Value string // Const value
	Doc   string // Documentation comment
}

// SharedAlias is a deprecated alias from a capability/host package to a type in
// the shared `types` package (e.g. type TrackInfo = types.Track). Def is the
// resolved shared struct, kept for XTP schema inlining.
type SharedAlias struct {
	Name   string    // local name, e.g. "TrackInfo"
	Target string    // alias target, e.g. "types.Track"
	Doc    string    // doc comment (carries the Deprecated: line)
	Def    StructDef // resolved shared struct shape
}

// ImportsSharedTypes reports whether this capability references the shared types package.
// A reference can come from a deprecated re-export alias (e.g. type SongRef = types.SongRef)
// or directly from the canonical form (e.g. types.SongRef) in a struct field or a method
// signature, so the generated Go import must be emitted even when no alias is declared.
func (c Capability) ImportsSharedTypes() bool {
	if len(c.SharedAliases) > 0 || structsReferenceSharedTypes(c.Structs) {
		return true
	}
	for _, m := range c.Methods {
		if typeReferencesSharedTypes(m.Input.Type) || typeReferencesSharedTypes(m.Output.Type) {
			return true
		}
	}
	return false
}

// ImportsSharedTypes reports whether this service references the shared types package.
func (s Service) ImportsSharedTypes() bool {
	if len(s.SharedAliases) > 0 || structsReferenceSharedTypes(s.Structs) {
		return true
	}
	for _, m := range s.Methods {
		for _, p := range m.Params {
			if typeReferencesSharedTypes(p.Type) {
				return true
			}
		}
		for _, r := range m.Returns {
			if typeReferencesSharedTypes(r.Type) {
				return true
			}
		}
	}
	return false
}

// structsReferenceSharedTypes reports whether any field across the given structs
// refers to the shared types package by its qualified name (e.g. types.SongRef,
// []types.SongRef, map[string]types.SongRef).
func structsReferenceSharedTypes(structs []StructDef) bool {
	for _, st := range structs {
		for _, f := range st.Fields {
			if typeReferencesSharedTypes(f.Type) {
				return true
			}
		}
	}
	return false
}

// typeReferencesSharedTypes reports whether a Go type expression refers to the
// shared types package by its qualified name, accounting for pointer, slice, and
// map wrappers (e.g. types.SongRef, []types.SongRef, map[string]types.SongRef).
func typeReferencesSharedTypes(goType string) bool {
	refs := map[string]bool{}
	collectReferencedTypes(goType, refs)
	for t := range refs {
		if strings.HasPrefix(t, sharedTypesPrefix) {
			return true
		}
	}
	return false
}

// KnownStructs returns a map of struct names defined in this capability,
// including shared-alias names so Rust field-type resolution finds them.
func (c Capability) KnownStructs() map[string]bool {
	result := make(map[string]bool)
	for _, st := range c.Structs {
		result[st.Name] = true
	}
	for _, sa := range c.SharedAliases {
		result[sa.Name] = true
	}
	return result
}

// Export represents an exported WASM function within a capability.
type Export struct {
	Name       string // Go method name (e.g., "GetArtistBiography")
	ExportName string // WASM export name (e.g., "nd_get_artist_biography")
	Input      Param  // Single input parameter (the struct type)
	Output     Param  // Single output return value (the struct type)
	Doc        string // Documentation comment for the method
}

// ProviderInterfaceName returns the optional provider interface name.
// For a method "GetArtistBiography", returns "ArtistBiographyProvider".
func (e Export) ProviderInterfaceName() string {
	// Remove "Get", "On", etc. prefixes and add "Provider" suffix
	name := e.Name
	for _, prefix := range []string{"Get", "On"} {
		if strings.HasPrefix(name, prefix) {
			name = name[len(prefix):]
			break
		}
	}
	return name + "Provider"
}

// ImplVarName returns the internal implementation variable name.
// For "GetArtistBiography", returns "artistBiographyImpl".
func (e Export) ImplVarName() string {
	name := e.Name
	for _, prefix := range []string{"Get", "On"} {
		if strings.HasPrefix(name, prefix) {
			name = name[len(prefix):]
			break
		}
	}
	// Convert to camelCase
	if len(name) > 0 {
		name = strings.ToLower(string(name[0])) + name[1:]
	}
	return name + "Impl"
}

// ExportFuncName returns the unexported WASM export function name.
// For "nd_get_artist_biography", returns "_ndGetArtistBiography".
func (e Export) ExportFuncName() string {
	// Convert snake_case to PascalCase
	parts := strings.Split(e.ExportName, "_")
	var result strings.Builder
	result.WriteString("_")
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(string(part[0])))
			result.WriteString(part[1:])
		}
	}
	return result.String()
}

// HasInput returns true if the method has an input parameter.
func (e Export) HasInput() bool {
	return e.Input.Type != ""
}

// HasOutput returns true if the method has a non-error return value.
func (e Export) HasOutput() bool {
	return e.Output.Type != ""
}

// IsPointerOutput returns true if the output type is a pointer.
func (e Export) IsPointerOutput() bool {
	return strings.HasPrefix(e.Output.Type, "*")
}

// StructDef represents a Go struct type definition.
type StructDef struct {
	Name   string     // Go struct name (e.g., "Library")
	Fields []FieldDef // Struct fields
	Doc    string     // Documentation comment
}

// FieldDef represents a field within a struct.
type FieldDef struct {
	Name      string // Go field name (e.g., "TotalSongs")
	Type      string // Go type (e.g., "int32", "*string", "[]User")
	JSONTag   string // JSON tag value (e.g., "totalSongs,omitempty")
	OmitEmpty bool   // Whether the field has omitempty tag
	Doc       string // Field documentation
}

// OutputFileName returns the generated file name for this service.
func (s Service) OutputFileName() string {
	return strings.ToLower(s.Name) + "_gen.go"
}

// ExportPrefix returns the prefix for exported host function names.
func (s Service) ExportPrefix() string {
	return strings.ToLower(s.Name)
}

// KnownStructs returns a map of struct names defined in this service,
// including shared-alias names so Rust field-type resolution finds them.
func (s Service) KnownStructs() map[string]bool {
	result := make(map[string]bool)
	for _, st := range s.Structs {
		result[st.Name] = true
	}
	for _, sa := range s.SharedAliases {
		result[sa.Name] = true
	}
	return result
}

// HasErrors returns true if any method in the service returns an error.
func (s Service) HasErrors() bool {
	for _, m := range s.Methods {
		if m.HasError {
			return true
		}
	}
	return false
}

// Method represents a host function method within a service.
type Method struct {
	Name       string  // Go method name (e.g., "Call")
	ExportName string  // Optional override for export name
	Params     []Param // Method parameters (excluding context.Context)
	Returns    []Param // Return values (excluding error)
	HasError   bool    // Whether the method returns an error
	Doc        string  // Documentation comment for the method
}

// FunctionName returns the Extism host function export name.
func (m Method) FunctionName(servicePrefix string) string {
	if m.ExportName != "" {
		return m.ExportName
	}
	return servicePrefix + "_" + strings.ToLower(m.Name)
}

// RequestTypeName returns the generated request type name (public, for host-side code).
func (m Method) RequestTypeName(serviceName string) string {
	return serviceName + m.Name + "Request"
}

// ResponseTypeName returns the generated response type name (public, for host-side code).
func (m Method) ResponseTypeName(serviceName string) string {
	return serviceName + m.Name + "Response"
}

// ClientRequestTypeName returns the generated request type name (private, for client/PDK code).
func (m Method) ClientRequestTypeName(serviceName string) string {
	return lowerFirst(serviceName) + m.Name + "Request"
}

// ClientResponseTypeName returns the generated response type name (private, for client/PDK code).
func (m Method) ClientResponseTypeName(serviceName string) string {
	return lowerFirst(serviceName) + m.Name + "Response"
}

// lowerFirst returns the string with the first letter lowercased.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// HasParams returns true if the method has input parameters.
func (m Method) HasParams() bool {
	return len(m.Params) > 0
}

// HasReturns returns true if the method has return values (excluding error).
func (m Method) HasReturns() bool {
	return len(m.Returns) > 0
}

// IsErrorOnly returns true if the method only returns an error (no data fields).
func (m Method) IsErrorOnly() bool {
	return m.HasError && !m.HasReturns()
}

// IsSingleReturn returns true if the method has exactly one return value (excluding error).
func (m Method) IsSingleReturn() bool {
	return len(m.Returns) == 1
}

// IsMultiReturn returns true if the method has multiple return values (excluding error).
func (m Method) IsMultiReturn() bool {
	return len(m.Returns) > 1
}

// IsOptionPattern returns true if the method returns (value, bool) where the bool
// indicates existence (named "exists", "ok", or "found"). This pattern is used to
// generate Option<T> in Rust instead of a tuple.
func (m Method) IsOptionPattern() bool {
	if len(m.Returns) != 2 {
		return false
	}
	if m.Returns[1].Type != "bool" {
		return false
	}
	// Only treat as option pattern if the first return has a meaningful value type
	// (not just a bool check like Has())
	if m.Returns[0].Type == "bool" {
		return false
	}
	name := strings.ToLower(m.Returns[1].Name)
	return name == "exists" || name == "ok" || name == "found"
}

// ReturnSignature returns the Go return type signature for the wrapper function.
// For error-only: "error"
// For single return with error: "(Type, error)"
// For single return no error: "Type"
// For multi return: "(Type1, Type2, ..., error)"
func (m Method) ReturnSignature() string {
	if m.IsErrorOnly() {
		return "error"
	}
	var parts []string
	for _, r := range m.Returns {
		parts = append(parts, r.Type)
	}
	if m.HasError {
		parts = append(parts, "error")
	}
	// Single return without error doesn't need parentheses
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// ZeroValues returns the zero value expressions for all return types (excluding error).
// Used for error return statements like "return "", false, err".
func (m Method) ZeroValues() string {
	var zeros []string
	for _, r := range m.Returns {
		zeros = append(zeros, zeroValue(r.Type))
	}
	return strings.Join(zeros, ", ")
}

// zeroValue returns the zero value for a Go type.
func zeroValue(typ string) string {
	switch {
	case typ == "string":
		return `""`
	case typ == "bool":
		return "false"
	case typ == "int", typ == "int8", typ == "int16", typ == "int32", typ == "int64",
		typ == "uint", typ == "uint8", typ == "uint16", typ == "uint32", typ == "uint64",
		typ == "float32", typ == "float64":
		return "0"
	case typ == "[]byte":
		return "nil"
	case strings.HasPrefix(typ, "[]"):
		return "nil"
	case strings.HasPrefix(typ, "map["):
		return "nil"
	case strings.HasPrefix(typ, "*"):
		return "nil"
	case typ == "any", typ == "interface{}":
		return "nil"
	default:
		// For custom struct types, return empty struct
		return typ + "{}"
	}
}

// Param represents a method parameter or return value.
type Param struct {
	Name     string // Parameter name
	Type     string // Go type (e.g., "string", "int32", "[]byte")
	JSONName string // JSON field name (camelCase)
}

// IsByteSlice returns true if the parameter type is []byte.
func (p Param) IsByteSlice() bool {
	return p.Type == "[]byte"
}

// IsByteSlice returns true if the field type is []byte.
func (f FieldDef) IsByteSlice() bool {
	return f.Type == "[]byte"
}

// HasByteFields returns true if any method params, returns, or struct fields use []byte.
func (s Service) HasByteFields() bool {
	for _, m := range s.Methods {
		for _, p := range m.Params {
			if p.IsByteSlice() {
				return true
			}
		}
		for _, r := range m.Returns {
			if r.IsByteSlice() {
				return true
			}
		}
	}
	for _, st := range s.Structs {
		for _, f := range st.Fields {
			if f.IsByteSlice() {
				return true
			}
		}
	}
	return false
}

// HasByteFields returns true if any capability struct fields use []byte.
func (c Capability) HasByteFields() bool {
	for _, st := range c.Structs {
		for _, f := range st.Fields {
			if f.IsByteSlice() {
				return true
			}
		}
	}
	return false
}

// NewParam creates a Param with auto-generated JSON name.
func NewParam(name, typ string) Param {
	return Param{
		Name:     name,
		Type:     typ,
		JSONName: toJSONName(name),
	}
}

// toJSONName converts a Go identifier to camelCase JSON field name.
// This matches Rust serde's rename_all = "camelCase" behavior.
// Examples: "ConnectionID" -> "connectionId", "NewConnectionID" -> "newConnectionId"
func toJSONName(name string) string {
	if name == "" {
		return ""
	}

	runes := []rune(name)
	result := make([]rune, 0, len(runes))

	for i, r := range runes {
		if i == 0 {
			// First character is always lowercase
			result = append(result, unicode.ToLower(r))
		} else if unicode.IsUpper(r) {
			// Check if this is part of an acronym (consecutive uppercase)
			// or a word boundary
			prevIsUpper := unicode.IsUpper(runes[i-1])
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

			if prevIsUpper && !nextIsLower {
				// Middle of an acronym - lowercase it
				result = append(result, unicode.ToLower(r))
			} else if prevIsUpper && nextIsLower {
				// End of acronym followed by lowercase - this starts a new word
				// Keep uppercase
				result = append(result, r)
			} else {
				// Regular word boundary - keep uppercase
				result = append(result, r)
			}
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}

// ToSnakeCase converts a PascalCase or camelCase string to snake_case.
// It handles consecutive uppercase letters correctly (e.g., "ScheduleID" -> "schedule_id").
func ToSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Add underscore before uppercase, but not if:
			// - Previous char was uppercase AND next char is uppercase or end of string
			//   (this handles acronyms like "ID" in "NewScheduleID")
			prevUpper := runes[i-1] >= 'A' && runes[i-1] <= 'Z'
			nextUpper := i+1 < len(runes) && runes[i+1] >= 'A' && runes[i+1] <= 'Z'
			atEnd := i+1 == len(runes)

			// Only skip underscore if we're in the middle of an acronym
			if !prevUpper || (!nextUpper && !atEnd) {
				result.WriteByte('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// ToRustType converts a Go type to its Rust equivalent.
func ToRustType(goType string) string {
	return ToRustTypeWithStructs(goType, nil)
}

// RustParamType returns the Rust type for a function parameter (uses &str for strings).
func RustParamType(goType string) string {
	if goType == "string" {
		return "&str"
	}
	return ToRustType(goType)
}

// RustDefaultValue returns the default value for a Rust type.
func RustDefaultValue(goType string) string {
	switch goType {
	case "string":
		return `String::new()`
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "0"
	case "float32", "float64":
		return "0.0"
	case "bool":
		return "false"
	default:
		if strings.HasPrefix(goType, "[]") {
			return "Vec::new()"
		}
		if strings.HasPrefix(goType, "map[") {
			return "std::collections::HashMap::new()"
		}
		if strings.HasPrefix(goType, "*") {
			return "None"
		}
		return "serde_json::Value::Null"
	}
}

// RustFunctionName returns the Rust function name for a method (snake_case).
// Uses just the method name without service prefix since the module provides namespacing.
func (m Method) RustFunctionName(_ string) string {
	return ToSnakeCase(m.Name)
}

// RustDocComment returns a properly formatted Rust doc comment.
// Each line of the input doc string is prefixed with "/// ".
func RustDocComment(doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, "/// "+line)
	}
	return strings.Join(result, "\n")
}

// RustType returns the Rust type for this parameter.
func (p Param) RustType() string {
	return ToRustType(p.Type)
}

// RustTypeWithStructs returns the Rust type using known struct names.
func (p Param) RustTypeWithStructs(knownStructs map[string]bool) string {
	return ToRustTypeWithStructs(p.Type, knownStructs)
}

// RustTypeWithShared returns the Rust type, resolving shared-alias names to their
// canonical nd_pdk_types::X crate path (e.g. a return of []Track where
// type Track = types.Track renders as Vec<nd_pdk_types::Track>).
func (p Param) RustTypeWithShared(knownStructs map[string]bool, shared map[string]string) string {
	return ToRustTypeWithShared(p.Type, knownStructs, shared)
}

// RustParamType returns the Rust type for this parameter when used as a function argument.
func (p Param) RustParamType() string {
	return RustParamType(p.Type)
}

// RustParamTypeWithStructs returns the Rust param type using known struct names.
func (p Param) RustParamTypeWithStructs(knownStructs map[string]bool) string {
	if p.Type == "string" {
		return "&str"
	}
	return ToRustTypeWithStructs(p.Type, knownStructs)
}

// RustParamTypeWithShared returns the Rust param type, resolving shared-alias
// names to their canonical nd_pdk_types::X crate path.
func (p Param) RustParamTypeWithShared(knownStructs map[string]bool, shared map[string]string) string {
	if p.Type == "string" {
		return "&str"
	}
	return ToRustTypeWithShared(p.Type, knownStructs, shared)
}

// RustName returns the snake_case Rust name for this parameter.
func (p Param) RustName() string {
	return ToSnakeCase(p.Name)
}

// NeedsToOwned returns true if the parameter needs .to_owned() when used.
func (p Param) NeedsToOwned() bool {
	return p.Type == "string"
}

// RustType returns the Rust type for this field, using known struct names.
func (f FieldDef) RustType(knownStructs map[string]bool) string {
	return ToRustTypeWithStructs(f.Type, knownStructs)
}

// RustName returns the snake_case Rust name for this field.
func (f FieldDef) RustName() string {
	return ToSnakeCase(f.Name)
}

// NeedsDefault returns true if the field needs #[serde(default)] attribute.
// This is true for fields with omitempty tag.
func (f FieldDef) NeedsDefault() bool {
	return f.OmitEmpty
}

// ToRustTypeWithStructs converts a Go type to its Rust equivalent,
// using known struct names instead of serde_json::Value.
func ToRustTypeWithStructs(goType string, knownStructs map[string]bool) string {
	return toRustType(goType, knownStructs, nil)
}

// ToRustTypeWithShared resolves shared-alias names to their canonical nd_pdk_types::X path.
func ToRustTypeWithShared(goType string, knownStructs map[string]bool, shared map[string]string) string {
	return toRustType(goType, knownStructs, shared)
}

func toRustType(goType string, knownStructs map[string]bool, shared map[string]string) string {
	// Handle pointer types
	if strings.HasPrefix(goType, "*") {
		inner := toRustType(goType[1:], knownStructs, shared)
		return "Option<" + inner + ">"
	}
	// Handle slice types
	if strings.HasPrefix(goType, "[]") {
		if goType == "[]byte" {
			return "Vec<u8>"
		}
		inner := toRustType(goType[2:], knownStructs, shared)
		return "Vec<" + inner + ">"
	}
	// Handle map types
	if strings.HasPrefix(goType, "map[") {
		// Extract key and value types from map[K]V
		rest := goType[4:] // Remove "map["
		depth := 1
		keyEnd := 0
		for i, r := range rest {
			if r == '[' {
				depth++
			} else if r == ']' {
				depth--
				if depth == 0 {
					keyEnd = i
					break
				}
			}
		}
		keyType := rest[:keyEnd]
		valueType := rest[keyEnd+1:]
		return "std::collections::HashMap<" + toRustType(keyType, knownStructs, shared) + ", " + toRustType(valueType, knownStructs, shared) + ">"
	}

	switch goType {
	case "string":
		return "String"
	case "int", "int32":
		return "i32"
	case "int64":
		return "i64"
	case "uint", "uint32":
		return "u32"
	case "uint64":
		return "u64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	case "bool":
		return "bool"
	case "interface{}", "any":
		return "serde_json::Value"
	default:
		// Qualified reference to the shared types crate (e.g. types.Track ->
		// nd_pdk_types::Track).
		if rest, ok := strings.CutPrefix(goType, sharedTypesPrefix); ok {
			return "nd_pdk_types::" + rest
		}
		// Resolve shared-alias names to their canonical nd_pdk_types:: path.
		if shared != nil {
			if t, ok := shared[goType]; ok {
				return t
			}
		}
		// Check if this is a known struct type
		if knownStructs != nil && knownStructs[goType] {
			return goType
		}
		// For unknown custom types, fall back to Value
		return "serde_json::Value"
	}
}
