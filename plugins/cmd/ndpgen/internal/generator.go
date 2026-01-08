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
// Uses private (lowercase) type names for request/response structs.
func clientFuncMap(svc Service) template.FuncMap {
	return template.FuncMap{
		"lower":            strings.ToLower,
		"title":            strings.Title,
		"exportName":       func(m Method) string { return m.FunctionName(svc.ExportPrefix()) },
		"requestType":      func(m Method) string { return m.ClientRequestTypeName(svc.Name) },
		"responseType":     func(m Method) string { return m.ClientResponseTypeName(svc.Name) },
		"formatDoc":        formatDoc,
		"mockReturnValues": mockReturnValues,
	}
}

// mockReturnValues generates the testify mock return value accessors for a method.
// For example: args.String(0), args.Bool(1), args.Error(2)
func mockReturnValues(m Method) string {
	var parts []string
	idx := 0

	for _, r := range m.Returns {
		parts = append(parts, mockAccessor(r.Type, idx))
		idx++
	}

	if m.HasError {
		parts = append(parts, fmt.Sprintf("args.Error(%d)", idx))
	}

	return strings.Join(parts, ", ")
}

// mockAccessor returns the testify mock accessor call for a given type and index.
func mockAccessor(typ string, idx int) string {
	switch {
	case typ == "string":
		return fmt.Sprintf("args.String(%d)", idx)
	case typ == "bool":
		return fmt.Sprintf("args.Bool(%d)", idx)
	case typ == "int":
		return fmt.Sprintf("args.Int(%d)", idx)
	case typ == "int64":
		return fmt.Sprintf("args.Get(%d).(int64)", idx)
	case typ == "int32":
		return fmt.Sprintf("args.Get(%d).(int32)", idx)
	case typ == "float64":
		return fmt.Sprintf("args.Get(%d).(float64)", idx)
	case typ == "float32":
		return fmt.Sprintf("args.Get(%d).(float32)", idx)
	case typ == "[]byte":
		return fmt.Sprintf("args.Get(%d).([]byte)", idx)
	default:
		// For slices, maps, pointers, and custom types, use Get with type assertion
		return fmt.Sprintf("args.Get(%d).(%s)", idx, typ)
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

// capabilityTemplateData holds data for capability template execution.
type capabilityTemplateData struct {
	Package    string
	Capability Capability
}

// capabilityFuncMap returns template functions for capability code generation.
func capabilityFuncMap(cap Capability) template.FuncMap {
	return template.FuncMap{
		"formatDoc":         formatDoc,
		"indent":            indentText,
		"agentName":         capabilityAgentName,
		"providerInterface": func(e Export) string { return e.ProviderInterfaceName() },
		"implVar":           func(e Export) string { return e.ImplVarName() },
		"exportFunc":        func(e Export) string { return e.ExportFuncName() },
	}
}

// indentText adds n tabs to each line of text.
func indentText(n int, s string) string {
	indent := strings.Repeat("\t", n)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// capabilityAgentName returns the interface name for a capability.
// Uses the Go interface name stripped of common suffixes.
func capabilityAgentName(cap Capability) string {
	name := cap.Interface
	// Remove common suffixes to get a clean name
	for _, suffix := range []string{"Agent", "Callback", "Service"} {
		if strings.HasSuffix(name, suffix) {
			name = name[:len(name)-len(suffix)]
			break
		}
	}
	// Use the shortened name or the original if no suffix found
	if name == "" {
		name = cap.Interface
	}
	return name
}

// GenerateCapabilityGo generates Go export wrapper code for a capability.
func GenerateCapabilityGo(cap Capability, pkgName string) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/capability.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading capability template: %w", err)
	}

	tmpl, err := template.New("capability").Funcs(capabilityFuncMap(cap)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := capabilityTemplateData{
		Package:    pkgName,
		Capability: cap,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateCapabilityGoStub generates stub code for non-WASM platforms.
func GenerateCapabilityGoStub(cap Capability, pkgName string) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/capability_stub.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading capability stub template: %w", err)
	}

	tmpl, err := template.New("capability_stub").Funcs(capabilityFuncMap(cap)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := capabilityTemplateData{
		Package:    pkgName,
		Capability: cap,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// rustCapabilityFuncMap returns template functions for Rust capability code generation.
func rustCapabilityFuncMap(cap Capability) template.FuncMap {
	knownStructs := cap.KnownStructs()
	return template.FuncMap{
		"rustDocComment":      RustDocComment,
		"rustTypeAlias":       rustTypeAlias,
		"rustConstType":       rustConstType,
		"rustConstName":       rustConstName,
		"rustFieldName":       func(name string) string { return ToSnakeCase(name) },
		"rustMethodName":      func(name string) string { return ToSnakeCase(name) },
		"fieldRustType":       func(f FieldDef) string { return f.RustType(knownStructs) },
		"rustOutputType":      rustOutputType,
		"isPrimitiveRust":     isPrimitiveRustType,
		"skipSerializingFunc": skipSerializingFunc,
		"hasHashMap":          hasHashMap,
		"agentName":           capabilityAgentName,
		"providerInterface":   func(e Export) string { return e.ProviderInterfaceName() },
		"registerMacroName":   func(name string) string { return registerMacroName(cap.Name, name) },
		"snakeCase":           ToSnakeCase,
		"indent": func(spaces int, s string) string {
			indent := strings.Repeat(" ", spaces)
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = indent + line
				}
			}
			return strings.Join(lines, "\n")
		},
	}
}

// rustTypeAlias converts a Go type to its Rust equivalent for type aliases.
// For string types used as error sentinels/constants, we use &'static str
// since Rust consts can't be heap-allocated String values.
func rustTypeAlias(goType string) string {
	switch goType {
	case "string":
		return "&'static str"
	case "int", "int32":
		return "i32"
	case "int64":
		return "i64"
	default:
		return goType
	}
}

// rustConstType converts a Go type to its Rust equivalent for const declarations.
// For String types, it returns &'static str since Rust consts can't be heap-allocated.
func rustConstType(goType string) string {
	switch goType {
	case "string", "String":
		return "&'static str"
	case "int", "int32":
		return "i32"
	case "int64":
		return "i64"
	default:
		return goType
	}
}

// rustOutputType converts a Go type to Rust for capability method signatures.
// It handles pointer types specially - for capability outputs, pointers become the base type
// (not Option<T>) because Rust's Result<T, Error> already provides optional semantics.
//
// TODO: Pointer to primitive types (e.g., *string, *int32) are not handled correctly.
// Currently "*string" returns "string" instead of "String". This would generate invalid
// Rust code. No current capability uses this pattern, but it should be fixed if needed.
func rustOutputType(goType string) string {
	// Strip pointer prefix - capability outputs use Result<T, Error> for optionality
	if strings.HasPrefix(goType, "*") {
		return goType[1:]
	}
	// Convert Go primitives to Rust primitives
	switch goType {
	case "bool":
		return "bool"
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
	}
	return goType
}

// isPrimitiveRustType returns true if the Go type maps to a Rust primitive type.
func isPrimitiveRustType(goType string) bool {
	// Strip pointer prefix first
	if strings.HasPrefix(goType, "*") {
		goType = goType[1:]
	}
	switch goType {
	case "bool", "string", "int", "int32", "int64", "float32", "float64":
		return true
	}
	return false
}

// rustConstName converts a Go const name to Rust convention (SCREAMING_SNAKE_CASE).
func rustConstName(name string) string {
	return strings.ToUpper(ToSnakeCase(name))
}

// skipSerializingFunc returns the appropriate skip_serializing_if function name.
func skipSerializingFunc(goType string) string {
	if strings.HasPrefix(goType, "*") || strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") {
		return "Option::is_none"
	}
	switch goType {
	case "string":
		return "String::is_empty"
	case "bool":
		return "std::ops::Not::not"
	default:
		return "Option::is_none"
	}
}

// hasHashMap returns true if any struct in the capability uses HashMap.
func hasHashMap(cap Capability) bool {
	for _, st := range cap.Structs {
		for _, f := range st.Fields {
			if strings.HasPrefix(f.Type, "map[") {
				return true
			}
		}
	}
	return false
}

// registerMacroName returns the macro name for registering an optional method.
// For package "websocket" and method "OnClose", returns "register_websocket_close".
func registerMacroName(pkg, name string) string {
	// Remove common prefixes from method name
	for _, prefix := range []string{"Get", "On"} {
		if strings.HasPrefix(name, prefix) {
			name = name[len(prefix):]
			break
		}
	}
	return "register_" + ToSnakeCase(pkg) + "_" + ToSnakeCase(name)
}

// GenerateCapabilityRust generates Rust export wrapper code for a capability.
func GenerateCapabilityRust(cap Capability) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/capability.rs.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading Rust capability template: %w", err)
	}

	tmpl, err := template.New("capability_rust").Funcs(rustCapabilityFuncMap(cap)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := capabilityTemplateData{
		Package:    cap.Name,
		Capability: cap,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateCapabilityRustLib generates the lib.rs file for the Rust capabilities crate.
func GenerateCapabilityRustLib(capabilities []Capability) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by ndpgen. DO NOT EDIT.\n\n")
	buf.WriteString("//! Navidrome Plugin Development Kit - Capability Wrappers\n")
	buf.WriteString("//!\n")
	buf.WriteString("//! This crate provides type definitions, traits, and registration macros\n")
	buf.WriteString("//! for implementing Navidrome plugin capabilities in Rust.\n\n")

	// Module declarations
	for _, cap := range capabilities {
		moduleName := ToSnakeCase(cap.Name)
		buf.WriteString(fmt.Sprintf("pub mod %s;\n", moduleName))
	}

	return buf.Bytes(), nil
}

// pdkFuncMap returns the template functions for PDK code generation.
func pdkFuncMap() template.FuncMap {
	return template.FuncMap{
		"firstSentence":       firstSentence,
		"paramList":           pdkParamList,
		"returnList":          pdkReturnList,
		"argList":             pdkArgList,
		"argListWithReceiver": pdkArgListWithReceiver,
		"mockReturns":         pdkMockReturns,
		"constValue":          pdkConstValue,
		"stubTypeUnderlying":  stubTypeUnderlying,
		"methodReceiver":      pdkMethodReceiver,
	}
}

// stubTypeUnderlying returns the appropriate stub type for non-WASM builds.
// For types that reference internal packages (like memory.Memory), returns "struct{}".
func stubTypeUnderlying(t PDKType) string {
	underlying := t.Underlying
	// If the underlying type references a package (contains a dot), use a stub struct
	if strings.Contains(underlying, ".") {
		return "struct{}"
	}
	// For simple types like int, int32, return as-is
	return underlying
}

// firstSentence returns the first sentence of a doc string, normalized to a single line.
func firstSentence(doc string) string {
	if doc == "" {
		return ""
	}
	// Normalize whitespace (replace newlines with spaces, collapse multiple spaces)
	doc = strings.Join(strings.Fields(doc), " ")

	// Find first period followed by space or end
	for i, r := range doc {
		if r == '.' && (i+1 >= len(doc) || doc[i+1] == ' ') {
			return doc[:i+1]
		}
	}
	return doc
}

// pdkParamList generates a parameter list string for function signature.
func pdkParamList(params []PDKParam) string {
	var parts []string
	for _, p := range params {
		if p.Name != "" {
			parts = append(parts, p.Name+" "+p.Type)
		} else {
			parts = append(parts, p.Type)
		}
	}
	return strings.Join(parts, ", ")
}

// pdkReturnList generates a return list string for function signature.
func pdkReturnList(returns []PDKReturn) string {
	if len(returns) == 0 {
		return ""
	}
	if len(returns) == 1 && returns[0].Name == "" {
		return " " + returns[0].Type
	}
	var parts []string
	for _, r := range returns {
		if r.Name != "" {
			parts = append(parts, r.Name+" "+r.Type)
		} else {
			parts = append(parts, r.Type)
		}
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// pdkArgList generates an argument list string for function call.
func pdkArgList(params []PDKParam) string {
	var parts []string
	for _, p := range params {
		if p.Name != "" {
			parts = append(parts, p.Name)
		} else {
			parts = append(parts, "_")
		}
	}
	return strings.Join(parts, ", ")
}

// pdkArgListWithReceiver generates an argument list that includes the receiver variable
// as the first argument to PDKMock.Called(). This allows tests to verify which instance
// a method was called on.
func pdkArgListWithReceiver(params []PDKParam, typeName string) string {
	// Use lowercase first letter of type name as receiver variable
	receiverVar := strings.ToLower(typeName[:1])
	parts := []string{receiverVar}
	for _, p := range params {
		if p.Name != "" {
			parts = append(parts, p.Name)
		} else {
			parts = append(parts, "_")
		}
	}
	return strings.Join(parts, ", ")
}

// pdkMethodReceiver generates the receiver declaration for a method.
// Example: "r *HTTPRequest" or "m Memory"
func pdkMethodReceiver(receiver, typeName string) string {
	receiverVar := strings.ToLower(typeName[:1])
	if strings.HasPrefix(receiver, "*") {
		return receiverVar + " *" + typeName
	}
	return receiverVar + " " + typeName
}

// pdkMockReturns generates the mock return accessors for a function.
func pdkMockReturns(returns []PDKReturn) string {
	var parts []string
	for i, r := range returns {
		parts = append(parts, mockAccessorForType(r.Type, i))
	}
	return strings.Join(parts, ", ")
}

// mockAccessorForType returns the testify mock accessor for a type.
func mockAccessorForType(typ string, idx int) string {
	switch typ {
	case "string":
		return fmt.Sprintf("args.String(%d)", idx)
	case "bool":
		return fmt.Sprintf("args.Bool(%d)", idx)
	case "int":
		return fmt.Sprintf("args.Int(%d)", idx)
	case "error":
		return fmt.Sprintf("args.Error(%d)", idx)
	case "[]byte":
		return fmt.Sprintf("args.Get(%d).([]byte)", idx)
	case "uint64":
		return fmt.Sprintf("args.Get(%d).(uint64)", idx)
	case "uint32":
		return fmt.Sprintf("args.Get(%d).(uint32)", idx)
	case "uint16":
		return fmt.Sprintf("args.Get(%d).(uint16)", idx)
	default:
		return fmt.Sprintf("args.Get(%d).(%s)", idx, typ)
	}
}

// pdkConstValue returns the value expression for a constant.
func pdkConstValue(c PDKConst) string {
	if c.Value == "" || c.Value == "iota" {
		return "iota"
	}
	return c.Value
}

// GeneratePDKGo generates the WASM implementation of the PDK wrapper package.
func GeneratePDKGo(symbols *PDKSymbols) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/pdk.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading pdk template: %w", err)
	}

	tmpl, err := template.New("pdk").Funcs(pdkFuncMap()).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, symbols); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GeneratePDKGoStub generates the native stub implementation of the PDK wrapper package.
func GeneratePDKGoStub(symbols *PDKSymbols) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/pdk_stub.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading pdk stub template: %w", err)
	}

	tmpl, err := template.New("pdk_stub").Funcs(pdkFuncMap()).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, symbols); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// GeneratePDKTypesStub generates the native type definitions for the PDK wrapper package.
func GeneratePDKTypesStub(symbols *PDKSymbols) ([]byte, error) {
	tmplContent, err := templatesFS.ReadFile("templates/types_stub.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("reading types stub template: %w", err)
	}

	tmpl, err := template.New("types_stub").Funcs(pdkFuncMap()).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, symbols); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}
