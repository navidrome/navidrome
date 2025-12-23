package internal

import (
	"strings"
)

// Service represents a parsed host service interface.
type Service struct {
	Name       string   // Service name from annotation (e.g., "SubsonicAPI")
	Permission string   // Manifest permission key (e.g., "subsonicapi")
	Interface  string   // Go interface name (e.g., "SubsonicAPIService")
	Methods    []Method // Methods marked with //nd:hostfunc
	Doc        string   // Documentation comment for the service
}

// OutputFileName returns the generated file name for this service.
func (s Service) OutputFileName() string {
	return strings.ToLower(s.Name) + "_gen.go"
}

// ExportPrefix returns the prefix for exported host function names.
func (s Service) ExportPrefix() string {
	return strings.ToLower(s.Name)
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

// IsSimple returns true if the type can be passed directly on the WASM stack
// without JSON serialization (primitive numeric types).
func (p Param) IsSimple() bool {
	return IsSimpleType(p.Type)
}

// IsPTR returns true if the type should be passed via memory pointer
// (strings, bytes, and complex types that need JSON serialization).
func (p Param) IsPTR() bool {
	return !p.IsSimple()
}

// ValueType returns the Extism ValueType constant name for this parameter.
func (p Param) ValueType() string {
	return GoTypeToValueType(p.Type)
}

// IsSimpleType returns true if a Go type can be passed directly on WASM stack.
func IsSimpleType(typ string) bool {
	switch typ {
	case "int32", "uint32", "int64", "uint64", "float32", "float64", "bool":
		return true
	default:
		return false
	}
}

// IsStringType returns true if the type is a string.
func IsStringType(typ string) bool {
	return typ == "string"
}

// IsBytesType returns true if the type is []byte.
func IsBytesType(typ string) bool {
	return typ == "[]byte"
}

// NeedsJSON returns true if the type requires JSON serialization.
func NeedsJSON(typ string) bool {
	if IsSimpleType(typ) || IsStringType(typ) || IsBytesType(typ) {
		return false
	}
	return true
}

// GoTypeToValueType returns the Extism ValueType constant for a Go type.
func GoTypeToValueType(typ string) string {
	switch typ {
	case "int32", "uint32":
		return "extism.ValueTypeI32"
	case "int64", "uint64":
		return "extism.ValueTypeI64"
	case "float32":
		return "extism.ValueTypeF32"
	case "float64":
		return "extism.ValueTypeF64"
	case "bool":
		return "extism.ValueTypeI32" // bool as i32
	default:
		// strings, []byte, structs, maps, slices all use PTR (i64)
		return "extism.ValueTypePTR"
	}
}

// AllParamsSimple returns true if all params can be passed on the stack.
func (m Method) AllParamsSimple() bool {
	for _, p := range m.Params {
		if !p.IsSimple() {
			return false
		}
	}
	return true
}

// AllReturnsSimple returns true if all returns can be passed on the stack.
func (m Method) AllReturnsSimple() bool {
	for _, r := range m.Returns {
		if !r.IsSimple() {
			return false
		}
	}
	return true
}

// NeedsRequestType returns true if a request struct is needed.
// Only needed when we have complex params that require JSON.
func (m Method) NeedsRequestType() bool {
	if !m.HasParams() {
		return false
	}
	for _, p := range m.Params {
		if NeedsJSON(p.Type) {
			return true
		}
	}
	return false
}

// NeedsResponseType returns true if a response struct is needed.
// Needed when we have complex returns that require JSON (but not for error-only methods).
func (m Method) NeedsResponseType() bool {
	// Error-only methods return a simple string, not JSON
	if m.IsErrorOnly() {
		return false
	}
	// If there's an error with other returns, we need a response type
	if m.HasError && m.HasReturns() {
		return true
	}
	for _, r := range m.Returns {
		if NeedsJSON(r.Type) {
			return true
		}
	}
	return false
}

// IsErrorOnly returns true if the method only returns an error (no other return values).
func (m Method) IsErrorOnly() bool {
	return m.HasError && !m.HasReturns()
}

// toJSONName converts a Go identifier to camelCase JSON field name.
func toJSONName(name string) string {
	if name == "" {
		return ""
	}
	// Simple conversion: lowercase first letter
	return strings.ToLower(name[:1]) + name[1:]
}
