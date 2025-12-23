package internal

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// GenerateService generates the host function wrapper code for a service.
func GenerateService(svc Service, pkgName string) ([]byte, error) {
	tmpl, err := template.New("service").Funcs(template.FuncMap{
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
	}).Parse(serviceTemplate)
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

type templateData struct {
	Package          string
	Service          Service
	NeedsJSON        bool
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

const serviceTemplate = `// Code generated by hostgen. DO NOT EDIT.

package {{.Package}}

import (
	"context"
{{- if .NeedsJSON}}
	"encoding/json"
{{- end}}

	extism "github.com/extism/go-sdk"
)

{{- /* Generate request/response types only when needed */ -}}
{{range .Service.Methods}}
{{- if needsRequestType .}}

// {{requestType .}} is the request type for {{$.Service.Name}}.{{.Name}}.
type {{requestType .}} struct {
{{- range .Params}}
	{{title .Name}} {{.Type}} ` + "`" + `json:"{{.JSONName}}"` + "`" + `
{{- end}}
}
{{- end}}
{{- if needsRespType .}}

// {{responseType .}} is the response type for {{$.Service.Name}}.{{.Name}}.
type {{responseType .}} struct {
{{- range .Returns}}
	{{title .Name}} {{.Type}} ` + "`" + `json:"{{.JSONName}},omitempty"` + "`" + `
{{- end}}
	Error string ` + "`" + `json:"error,omitempty"` + "`" + `
}
{{- end}}
{{end}}

// Register{{.Service.Name}}HostFunctions registers {{.Service.Name}} service host functions.
// The returned host functions should be added to the plugin's configuration.
func Register{{.Service.Name}}HostFunctions(service {{.Service.Interface}}) []extism.HostFunction {
	return []extism.HostFunction{
{{- range .Service.Methods}}
		new{{$.Service.Name}}{{.Name}}HostFunction(service),
{{- end}}
	}
}
{{range .Service.Methods}}

func new{{$.Service.Name}}{{.Name}}HostFunction(service {{$.Service.Interface}}) extism.HostFunction {
	return extism.NewHostFunctionWithStack(
		"{{exportName .}}",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
{{- if .HasParams}}
{{- if needsRequestType .}}
			// Read JSON request from plugin memory
			reqBytes, err := p.ReadBytes(stack[0])
			if err != nil {
				{{$.Service.Name | lower}}WriteError(p, stack, err)
				return
			}
			var req {{requestType .}}
			if err := json.Unmarshal(reqBytes, &req); err != nil {
				{{$.Service.Name | lower}}WriteError(p, stack, err)
				return
			}
{{- else}}
			// Read parameters from stack
{{- range $i, $p := .Params}}
			{{readParam $p $i}}
{{- end}}
{{- end}}
{{- end}}

			// Call the service method
{{- $m := .}}
{{- if .HasReturns}}
{{- if .HasError}}
			{{range $i, $r := .Returns}}{{if $i}}, {{end}}{{lower $r.Name}}{{end}}, err := service.{{.Name}}(ctx{{range .Params}}, {{if needsRequestType $m}}req.{{title .Name}}{{else}}{{.Name}}{{end}}{{end}})
{{- else}}
			{{range $i, $r := .Returns}}{{if $i}}, {{end}}{{lower $r.Name}}{{end}} := service.{{.Name}}(ctx{{range .Params}}, {{if needsRequestType $m}}req.{{title .Name}}{{else}}{{.Name}}{{end}}{{end}})
{{- end}}
{{- else if .HasError}}
			err {{if hasErrFromRead .}}={{else}}:={{end}} service.{{.Name}}(ctx{{range .Params}}, {{if needsRequestType $m}}req.{{title .Name}}{{else}}{{.Name}}{{end}}{{end}})
{{- else}}
			service.{{.Name}}(ctx{{range .Params}}, {{if needsRequestType $m}}req.{{title .Name}}{{else}}{{.Name}}{{end}}{{end}})
{{- end}}
{{- if .HasError}}
			if err != nil {
{{- if isErrorOnly .}}
				// Write error string to plugin memory
				if ptr, err := p.WriteString(err.Error()); err == nil {
					stack[0] = ptr
				}
{{- else if needsRespType .}}
				{{$.Service.Name | lower}}WriteError(p, stack, err)
{{- end}}
				return
			}
{{- end}}

{{- if isErrorOnly .}}
			// Write empty string to indicate success
			if ptr, err := p.WriteString(""); err == nil {
				stack[0] = ptr
			}
{{- else if needsRespType .}}
			// Write JSON response to plugin memory
			resp := {{responseType .}}{
{{- range .Returns}}
				{{title .Name}}: {{lower .Name}},
{{- end}}
			}
			{{$.Service.Name | lower}}WriteResponse(p, stack, resp)
{{- else if .HasReturns}}
			// Write return values to stack
{{- range $i, $r := .Returns}}
			{{writeReturn $r $i (lower $r.Name)}}
{{- end}}
{{- end}}
		},
{{- if needsRequestType $m}}
		[]extism.ValueType{extism.ValueTypePTR},
{{- else}}
		[]extism.ValueType{ {{- range $i, $p := .Params}}{{if $i}}, {{end}}{{valueType $p.Type}}{{end}}{{if not .HasParams}}{{end}} },
{{- end}}
{{- if or (needsRespType .) (isErrorOnly .)}}
		[]extism.ValueType{extism.ValueTypePTR},
{{- else}}
		[]extism.ValueType{ {{- range $i, $r := .Returns}}{{if $i}}, {{end}}{{valueType $r.Type}}{{end}}{{if not .HasReturns}}{{end}} },
{{- end}}
	)
}
{{end}}
{{- if .NeedsWriteHelper}}

// {{.Service.Name | lower}}WriteResponse writes a JSON response to plugin memory.
func {{.Service.Name | lower}}WriteResponse(p *extism.CurrentPlugin, stack []uint64, resp any) {
	respBytes, err := json.Marshal(resp)
	if err != nil {
		{{.Service.Name | lower}}WriteError(p, stack, err)
		return
	}
	respPtr, err := p.WriteBytes(respBytes)
	if err != nil {
		stack[0] = 0
		return
	}
	stack[0] = respPtr
}
{{- end}}
{{- if .NeedsErrorHelper}}

// {{.Service.Name | lower}}WriteError writes an error response to plugin memory.
func {{.Service.Name | lower}}WriteError(p *extism.CurrentPlugin, stack []uint64, err error) {
	errResp := struct {
		Error string ` + "`" + `json:"error"` + "`" + `
	}{Error: err.Error()}
	respBytes, _ := json.Marshal(errResp)
	respPtr, _ := p.WriteBytes(respBytes)
	stack[0] = respPtr
}
{{- end}}
`
