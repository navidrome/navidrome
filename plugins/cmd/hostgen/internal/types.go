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

// toJSONName converts a Go identifier to camelCase JSON field name.
func toJSONName(name string) string {
	if name == "" {
		return ""
	}
	// Simple conversion: lowercase first letter
	return strings.ToLower(name[:1]) + name[1:]
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
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
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
