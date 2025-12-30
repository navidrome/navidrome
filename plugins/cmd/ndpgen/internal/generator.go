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
		"lower":        strings.ToLower,
		"title":        strings.Title,
		"exportName":   func(m Method) string { return m.FunctionName(svc.ExportPrefix()) },
		"requestType":  func(m Method) string { return m.RequestTypeName(svc.Name) },
		"responseType": func(m Method) string { return m.ResponseTypeName(svc.Name) },
	}
}

// clientFuncMap returns the template functions for client code generation.
func clientFuncMap(svc Service) template.FuncMap {
	return template.FuncMap{
		"lower":        strings.ToLower,
		"title":        strings.Title,
		"exportName":   func(m Method) string { return m.FunctionName(svc.ExportPrefix()) },
		"requestType":  func(m Method) string { return m.RequestTypeName(svc.Name) },
		"responseType": func(m Method) string { return m.ResponseTypeName(svc.Name) },
		"formatDoc":    formatDoc,
	}
}

// pythonFuncMap returns the template functions for Python client code generation.
func pythonFuncMap(svc Service) template.FuncMap {
	return template.FuncMap{
		"lower":            strings.ToLower,
		"exportName":       func(m Method) string { return m.FunctionName(svc.ExportPrefix()) },
		"pythonFunc":       func(m Method) string { return m.PythonFunctionName(svc.ExportPrefix()) },
		"pythonResultType": func(m Method) string { return m.PythonResultTypeName(svc.Name) },
		"pythonDefault":    pythonDefaultValue,
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
		Package: pkgName,
		Service: svc,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateClientGo generates client wrapper code for plugins to call host functions.
func GenerateClientGo(svc Service, pkgName string) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/client.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading client template: %w", err)
	}

	tmpl, err := template.New("client").Funcs(clientFuncMap(svc)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := templateData{
		Package: pkgName,
		Service: svc,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateClientGoStub generates stub code for non-WASM platforms.
// These stubs provide type definitions and function signatures for IDE support,
// but panic at runtime since host functions are only available in WASM plugins.
func GenerateClientGoStub(svc Service, pkgName string) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/client_stub.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading client stub template: %w", err)
	}

	tmpl, err := template.New("client_stub").Funcs(clientFuncMap(svc)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := templateData{
		Package: pkgName,
		Service: svc,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

type templateData struct {
	Package string
	Service Service
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

// GenerateClientPython generates Python client wrapper code for plugins.
func GenerateClientPython(svc Service) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/client.py.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading Python client template: %w", err)
	}

	tmpl, err := template.New("client_py").Funcs(pythonFuncMap(svc)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := templateData{
		Service: svc,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// pythonDefaultValue returns a Python default value for response.get() calls.
func pythonDefaultValue(p Param) string {
	switch p.Type {
	case "string":
		return `, ""`
	case "int", "int32", "int64":
		return ", 0"
	case "float32", "float64":
		return ", 0.0"
	case "bool":
		return ", False"
	case "[]byte":
		return ", b\"\""
	default:
		return ", None"
	}
}

// rustFuncMap returns the template functions for Rust client code generation.
func rustFuncMap(svc Service) template.FuncMap {
	knownStructs := svc.KnownStructs()
	return template.FuncMap{
		"lower":          strings.ToLower,
		"exportName":     func(m Method) string { return m.FunctionName(svc.ExportPrefix()) },
		"requestType":    func(m Method) string { return m.RequestTypeName(svc.Name) },
		"responseType":   func(m Method) string { return m.ResponseTypeName(svc.Name) },
		"rustFunc":       func(m Method) string { return m.RustFunctionName(svc.ExportPrefix()) },
		"rustDocComment": RustDocComment,
		"rustType":       func(p Param) string { return p.RustTypeWithStructs(knownStructs) },
		"rustParamType":  func(p Param) string { return p.RustParamTypeWithStructs(knownStructs) },
		"fieldRustType":  func(f FieldDef) string { return f.RustType(knownStructs) },
	}
}

// GenerateClientRust generates Rust client wrapper code for plugins.
func GenerateClientRust(svc Service) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/client.rs.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading Rust client template: %w", err)
	}

	tmpl, err := template.New("client_rs").Funcs(rustFuncMap(svc)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := templateData{
		Service: svc,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// firstLine returns the first line of a multi-line string, with the first word removed.
func firstLine(s string) string {
	line := s
	if idx := strings.Index(s, "\n"); idx >= 0 {
		line = s[:idx]
	}
	// Remove the first word (service name like "ArtworkService")
	if idx := strings.Index(line, " "); idx >= 0 {
		line = line[idx+1:]
	}
	return line
}

// GenerateRustLib generates the lib.rs file that exposes all service modules.
func GenerateRustLib(services []Service) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/lib.rs.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading Rust lib template: %w", err)
	}

	tmpl, err := template.New("lib_rs").Funcs(template.FuncMap{
		"lower":     strings.ToLower,
		"firstLine": firstLine,
	}).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := struct {
		Services []Service
	}{
		Services: services,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateGoDoc generates the doc.go file that provides package documentation.
func GenerateGoDoc(services []Service, pkgName string) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/doc.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading Go doc template: %w", err)
	}

	tmpl, err := template.New("doc_go").Funcs(template.FuncMap{
		"firstLine": firstLine,
	}).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := struct {
		Package  string
		Services []Service
	}{
		Package:  pkgName,
		Services: services,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateGoMod generates the go.mod file for the Go client library.
func GenerateGoMod() ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/go.mod.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading go.mod template: %w", err)
	}
	return tmplContent, nil
}
