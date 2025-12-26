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
