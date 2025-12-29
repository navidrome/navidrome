package internal

import (
	"strings"
	"unicode"
)

// Service represents a parsed host service interface.
type Service struct {
	Name       string      // Service name from annotation (e.g., "SubsonicAPI")
	Permission string      // Manifest permission key (e.g., "subsonicapi")
	Interface  string      // Go interface name (e.g., "SubsonicAPIService")
	Methods    []Method    // Methods marked with //nd:hostfunc
	Doc        string      // Documentation comment for the service
	Structs    []StructDef // Structs used by this service
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

// KnownStructs returns a map of struct names defined in this service.
func (s Service) KnownStructs() map[string]bool {
	result := make(map[string]bool)
	for _, st := range s.Structs {
		result[st.Name] = true
	}
	return result
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

// RequestTypeName returns the generated request type name.
func (m Method) RequestTypeName(serviceName string) string {
	return serviceName + m.Name + "Request"
}

// ResponseTypeName returns the generated response type name.
func (m Method) ResponseTypeName(serviceName string) string {
	return serviceName + m.Name + "Response"
}

// HasParams returns true if the method has input parameters.
func (m Method) HasParams() bool {
	return len(m.Params) > 0
}

// HasReturns returns true if the method has return values (excluding error).
func (m Method) HasReturns() bool {
	return len(m.Returns) > 0
}

// Param represents a method parameter or return value.
type Param struct {
	Name     string // Parameter name
	Type     string // Go type (e.g., "string", "int32", "[]byte")
	JSONName string // JSON field name (camelCase)
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

// ToPythonType converts a Go type to its Python equivalent.
func ToPythonType(goType string) string {
	switch goType {
	case "string":
		return "str"
	case "int", "int32", "int64":
		return "int"
	case "float32", "float64":
		return "float"
	case "bool":
		return "bool"
	case "[]byte":
		return "bytes"
	default:
		return "Any"
	}
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

// PythonFunctionName returns the Python function name for a method.
func (m Method) PythonFunctionName(servicePrefix string) string {
	return ToSnakeCase(servicePrefix + m.Name)
}

// PythonResultTypeName returns the Python dataclass name for multi-value returns.
func (m Method) PythonResultTypeName(serviceName string) string {
	return serviceName + m.Name + "Result"
}

// NeedsResultClass returns true if the method needs a dataclass for returns.
func (m Method) NeedsResultClass() bool {
	return len(m.Returns) > 1
}

// PythonType returns the Python type for this parameter.
func (p Param) PythonType() string {
	return ToPythonType(p.Type)
}

// PythonName returns the snake_case Python name for this parameter.
func (p Param) PythonName() string {
	return ToSnakeCase(p.Name)
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
	case "int", "int32":
		return "0"
	case "int64":
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
	// Handle pointer types
	if strings.HasPrefix(goType, "*") {
		inner := ToRustTypeWithStructs(goType[1:], knownStructs)
		return "Option<" + inner + ">"
	}
	// Handle slice types
	if strings.HasPrefix(goType, "[]") {
		if goType == "[]byte" {
			return "Vec<u8>"
		}
		inner := ToRustTypeWithStructs(goType[2:], knownStructs)
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
		return "std::collections::HashMap<" + ToRustTypeWithStructs(keyType, knownStructs) + ", " + ToRustTypeWithStructs(valueType, knownStructs) + ">"
	}

	switch goType {
	case "string":
		return "String"
	case "int", "int32":
		return "i32"
	case "int64":
		return "i64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	case "bool":
		return "bool"
	case "interface{}", "any":
		return "serde_json::Value"
	default:
		// Check if this is a known struct type
		if knownStructs != nil && knownStructs[goType] {
			return goType
		}
		// For unknown custom types, fall back to Value
		return "serde_json::Value"
	}
}
