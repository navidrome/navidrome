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
