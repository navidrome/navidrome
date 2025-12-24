package internal

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// hostFuncMap returns the template functions for host code generation.
func hostFuncMap(svc Service) template.FuncMap {
	return template.FuncMap{
		"lower":            strings.ToLower,
		"title":            strings.Title,
		"exportName":       func(m Method) string { return m.FunctionName(svc.ExportPrefix()) },
		"requestType":      func(m Method) string { return m.RequestTypeName(svc.Name) },
		"responseType":     func(m Method) string { return m.ResponseTypeName(svc.Name) },
		"valueType":        GoTypeToValueType,
		"isSimple":         IsSimpleType,
		"isString":         IsStringType,
		"isBytes":          IsBytesType,
		"needsJSON":        NeedsJSON,
		"needsRequestType": func(m Method) bool { return m.NeedsRequestType() },
		"needsRespType":    func(m Method) bool { return m.NeedsResponseType() },
		"isErrorOnly":      func(m Method) bool { return m.IsErrorOnly() },
		"hasErrFromRead":   hasErrorFromRead,
		"readParam":        generateReadParam,
		"writeReturn":      generateWriteReturn,
		"encodeReturn":     generateEncodeReturn,
	}
}

// clientFuncMap returns the template functions for client code generation.
func clientFuncMap(svc Service) template.FuncMap {
	return template.FuncMap{
		"lower":             strings.ToLower,
		"title":             strings.Title,
		"exportName":        func(m Method) string { return m.FunctionName(svc.ExportPrefix()) },
		"responseType":      func(m Method) string { return m.ResponseTypeName(svc.Name) },
		"isSimple":          IsSimpleType,
		"isString":          IsStringType,
		"isBytes":           IsBytesType,
		"needsJSON":         NeedsJSON,
		"needsRespType":     func(m Method) bool { return m.NeedsResponseType() },
		"isErrorOnly":       func(m Method) bool { return m.IsErrorOnly() },
		"wasmParamType":     wasmParamType,
		"wasmReturnType":    wasmReturnType,
		"wrapperReturnType": func(m Method, svcName string) string { return wrapperReturnType(m, svcName) },
		"clientCallArg":     clientCallArg,
		"decodeResult":      decodeResult,
		"formatDoc":         formatDoc,
	}
}

// GenerateHost generates the host function wrapper code for a service.
func GenerateHost(svc Service, pkgName string) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/host.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading host template: %w", err)
	}

	tmpl, err := template.New("host").Funcs(hostFuncMap(svc)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := templateData{
		Package:          pkgName,
		Service:          svc,
		NeedsJSON:        serviceNeedsJSON(svc),
		NeedsWriteHelper: serviceNeedsWriteHelper(svc),
		NeedsErrorHelper: serviceNeedsErrorHelper(svc),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateService generates the host function wrapper code for a service.
// Deprecated: Use GenerateHost instead.
func GenerateService(svc Service, pkgName string) ([]byte, error) {
	return GenerateHost(svc, pkgName)
}

// GenerateClientGo generates client wrapper code for plugins to call host functions.
func GenerateClientGo(svc Service) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/client_go.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading client template: %w", err)
	}

	tmpl, err := template.New("client").Funcs(clientFuncMap(svc)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := templateData{
		Service:     svc,
		NeedsJSON:   serviceClientNeedsJSON(svc),
		NeedsErrors: serviceClientNeedsErrors(svc),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

type templateData struct {
	Package          string
	Service          Service
	NeedsJSON        bool
	NeedsErrors      bool // Client: needs "errors" import
	NeedsWriteHelper bool
	NeedsErrorHelper bool
}

// serviceNeedsJSON returns true if any method needs JSON encoding.
func serviceNeedsJSON(svc Service) bool {
	for _, m := range svc.Methods {
		for _, p := range m.Params {
			if NeedsJSON(p.Type) {
				return true
			}
		}
		for _, r := range m.Returns {
			if NeedsJSON(r.Type) {
				return true
			}
		}
		// Error responses are also JSON
		if m.HasError && m.NeedsResponseType() {
			return true
		}
	}
	return false
}

// serviceNeedsWriteHelper returns true if any method needs the write helper.
func serviceNeedsWriteHelper(svc Service) bool {
	for _, m := range svc.Methods {
		if m.NeedsResponseType() {
			return true
		}
	}
	return false
}

// serviceClientNeedsJSON returns true if any method needs JSON encoding in client code.
// This is true if any method has a response type (complex returns) or if any param/return needs JSON.
func serviceClientNeedsJSON(svc Service) bool {
	for _, m := range svc.Methods {
		// Response types use JSON for serialization
		if m.NeedsResponseType() {
			return true
		}
		// Parameters that need JSON marshaling
		for _, p := range m.Params {
			if NeedsJSON(p.Type) {
				return true
			}
		}
	}
	return false
}

// serviceClientNeedsErrors returns true if any method needs the errors package in client code.
// This is true if any method returns an error.
func serviceClientNeedsErrors(svc Service) bool {
	for _, m := range svc.Methods {
		if m.HasError {
			return true
		}
	}
	return false
}

// serviceNeedsErrorHelper returns true if any method needs error handling with JSON.
func serviceNeedsErrorHelper(svc Service) bool {
	for _, m := range svc.Methods {
		if m.HasError && m.NeedsResponseType() {
			return true
		}
	}
	return false
}

// hasErrorFromRead returns true if reading params declares an err variable.
// This happens when using needsRequestType (JSON) or when any param is a PTR type.
func hasErrorFromRead(m Method) bool {
	if m.NeedsRequestType() {
		return true
	}
	for _, p := range m.Params {
		if IsStringType(p.Type) || IsBytesType(p.Type) {
			return true
		}
	}
	return false
}

// generateReadParam generates code to read a parameter from the stack.
func generateReadParam(p Param, stackIndex int) string {
	switch {
	case IsSimpleType(p.Type):
		return generateReadSimple(p, stackIndex)
	case IsStringType(p.Type):
		return fmt.Sprintf(`%s, err := p.ReadString(stack[%d])
			if err != nil {
				return
			}`, p.Name, stackIndex)
	case IsBytesType(p.Type):
		return fmt.Sprintf(`%s, err := p.ReadBytes(stack[%d])
			if err != nil {
				return
			}`, p.Name, stackIndex)
	default:
		// Complex type - JSON
		return fmt.Sprintf(`%sBytes, err := p.ReadBytes(stack[%d])
			if err != nil {
				return
			}
			var %s %s
			if err := json.Unmarshal(%sBytes, &%s); err != nil {
				return
			}`, p.Name, stackIndex, p.Name, p.Type, p.Name, p.Name)
	}
}

// generateReadSimple generates code to read a simple type from the stack.
func generateReadSimple(p Param, stackIndex int) string {
	switch p.Type {
	case "int32":
		return fmt.Sprintf(`%s := extism.DecodeI32(stack[%d])`, p.Name, stackIndex)
	case "uint32":
		return fmt.Sprintf(`%s := extism.DecodeU32(stack[%d])`, p.Name, stackIndex)
	case "int64":
		return fmt.Sprintf(`%s := int64(stack[%d])`, p.Name, stackIndex)
	case "uint64":
		return fmt.Sprintf(`%s := stack[%d]`, p.Name, stackIndex)
	case "float32":
		return fmt.Sprintf(`%s := extism.DecodeF32(stack[%d])`, p.Name, stackIndex)
	case "float64":
		return fmt.Sprintf(`%s := extism.DecodeF64(stack[%d])`, p.Name, stackIndex)
	case "bool":
		return fmt.Sprintf(`%s := extism.DecodeI32(stack[%d]) != 0`, p.Name, stackIndex)
	default:
		return fmt.Sprintf(`// FIXME: unsupported type: %s`, p.Type)
	}
}

// generateWriteReturn generates code to write a return value to the stack.
func generateWriteReturn(p Param, stackIndex int, varName string) string {
	switch {
	case IsSimpleType(p.Type):
		return fmt.Sprintf(`stack[%d] = %s`, stackIndex, generateEncodeReturn(p, varName))
	case IsStringType(p.Type):
		return fmt.Sprintf(`if ptr, err := p.WriteString(%s); err == nil {
				stack[%d] = ptr
			}`, varName, stackIndex)
	case IsBytesType(p.Type):
		return fmt.Sprintf(`if ptr, err := p.WriteBytes(%s); err == nil {
				stack[%d] = ptr
			}`, varName, stackIndex)
	default:
		// Complex type - JSON
		return fmt.Sprintf(`if bytes, err := json.Marshal(%s); err == nil {
				if ptr, err := p.WriteBytes(bytes); err == nil {
					stack[%d] = ptr
				}
			}`, varName, stackIndex)
	}
}

// generateEncodeReturn generates the encoding expression for a simple return.
func generateEncodeReturn(p Param, varName string) string {
	switch p.Type {
	case "int32":
		return fmt.Sprintf("extism.EncodeI32(%s)", varName)
	case "uint32":
		return fmt.Sprintf("extism.EncodeU32(%s)", varName)
	case "int64":
		return fmt.Sprintf("uint64(%s)", varName)
	case "uint64":
		return varName
	case "float32":
		return fmt.Sprintf("extism.EncodeF32(%s)", varName)
	case "float64":
		return fmt.Sprintf("extism.EncodeF64(%s)", varName)
	case "bool":
		return fmt.Sprintf("func() uint64 { if %s { return 1 }; return 0 }()", varName)
	default:
		return "0"
	}
}

// Client-side helper functions for template

// wasmParamType returns the WASM parameter type for a Go parameter.
func wasmParamType(p Param) string {
	if IsSimpleType(p.Type) {
		return p.Type
	}
	// All pointer types (string, []byte, complex) use uint64 offset
	return "uint64"
}

// wasmReturnType returns the WASM return type declaration for a method.
func wasmReturnType(m Method) string {
	// Methods with JSON responses or error-only return uint64 (pointer)
	if m.NeedsResponseType() || m.IsErrorOnly() {
		return "uint64"
	}
	// Simple return types
	if len(m.Returns) == 1 && IsSimpleType(m.Returns[0].Type) {
		return m.Returns[0].Type
	}
	// No returns or multiple returns - use uint64 for pointer
	if len(m.Returns) == 0 {
		return ""
	}
	return "uint64"
}

// wrapperReturnType returns the Go return type for the wrapper function.
func wrapperReturnType(m Method, svcName string) string {
	if m.NeedsResponseType() {
		return fmt.Sprintf("(*%s%sResponse, error)", svcName, m.Name)
	}
	if m.IsErrorOnly() {
		return "error"
	}
	if len(m.Returns) == 1 {
		return m.Returns[0].Type
	}
	return ""
}

// clientCallArg returns the argument expression for calling the host function.
func clientCallArg(p Param) string {
	if IsSimpleType(p.Type) {
		return p.Name
	}
	// Pointer types use .Offset()
	return p.Name + "Mem.Offset()"
}

// decodeResult generates code to decode a simple return value.
func decodeResult(p Param, varName string) string {
	switch p.Type {
	case "int32":
		return fmt.Sprintf("int32(%s)", varName)
	case "uint32":
		return fmt.Sprintf("uint32(%s)", varName)
	case "int64":
		return fmt.Sprintf("int64(%s)", varName)
	case "uint64":
		return varName
	case "float32":
		return fmt.Sprintf("math.Float32frombits(uint32(%s))", varName)
	case "float64":
		return fmt.Sprintf("math.Float64frombits(%s)", varName)
	case "bool":
		return fmt.Sprintf("%s != 0", varName)
	case "string":
		// pdk.FindMemory returns a value type, ReadBytes has pointer receiver
		return fmt.Sprintf("func() string { m := pdk.FindMemory(%s); return string(m.ReadBytes()) }()", varName)
	case "[]byte":
		return fmt.Sprintf("func() []byte { m := pdk.FindMemory(%s); return m.ReadBytes() }()", varName)
	default:
		return varName
	}
}

// formatDoc formats a documentation string for Go comments.
// It prefixes each line with "// " and trims trailing whitespace.
func formatDoc(doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(doc), "\n")
	var result []string
	for _, line := range lines {
		result = append(result, "// "+strings.TrimRight(line, " \t"))
	}
	return strings.Join(result, "\n")
}
