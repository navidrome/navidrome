package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Annotation patterns
var (
	// //nd:hostservice name=ServiceName permission=key
	hostServicePattern = regexp.MustCompile(`//nd:hostservice\s+(.*)`)
	// //nd:hostfunc [name=CustomName]
	hostFuncPattern = regexp.MustCompile(`//nd:hostfunc(?:\s+(.*))?`)
	// //nd:capability name=PackageName [required=true]
	capabilityPattern = regexp.MustCompile(`//nd:capability\s+(.*)`)
	// //nd:export name=ExportName
	exportPattern = regexp.MustCompile(`//nd:export\s+(.*)`)
	// key=value pairs
	keyValuePattern = regexp.MustCompile(`(\w+)=(\S+)`)
)

// ParseDirectory parses all Go source files in a directory and extracts host services.
func ParseDirectory(dir string) ([]Service, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var services []Service
	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		// Skip generated files and test files
		if strings.HasSuffix(entry.Name(), "_gen.go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		parsed, err := parseFile(fset, path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		services = append(services, parsed...)
	}

	return services, nil
}

// ParseCapabilities parses all Go source files in a directory and extracts capabilities.
func ParseCapabilities(dir string) ([]Capability, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	fset := token.NewFileSet()

	// First pass: collect all structs and type aliases from all files in the package
	sharedStructMap := make(map[string]StructDef)
	sharedAliasMap := make(map[string]TypeAlias)
	var allConstGroups []ConstGroup

	var goFiles []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		// Skip generated files, test files, and doc.go
		if strings.HasSuffix(entry.Name(), "_gen.go") ||
			strings.HasSuffix(entry.Name(), "_test.go") ||
			entry.Name() == "doc.go" {
			continue
		}
		goFiles = append(goFiles, filepath.Join(dir, entry.Name()))
	}

	for _, path := range goFiles {
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parsing %s for types: %w", filepath.Base(path), err)
		}
		for _, s := range parseStructs(f) {
			sharedStructMap[s.Name] = s
		}
		for _, a := range parseTypeAliases(f) {
			sharedAliasMap[a.Name] = a
		}
		allConstGroups = append(allConstGroups, parseConstGroups(f)...)
	}

	// Second pass: parse capabilities using the shared type maps
	var capabilities []Capability
	for _, path := range goFiles {
		parsed, err := parseCapabilityFile(fset, path, sharedStructMap, sharedAliasMap, allConstGroups)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", filepath.Base(path), err)
		}
		capabilities = append(capabilities, parsed...)
	}

	return capabilities, nil
}

// parseCapabilityFile parses a single Go source file and extracts capabilities.
func parseCapabilityFile(fset *token.FileSet, path string, structMap map[string]StructDef, aliasMap map[string]TypeAlias, allConstGroups []ConstGroup) ([]Capability, error) {
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var capabilities []Capability

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			// Check for //nd:capability annotation in doc comment
			docText, rawDoc := getDocComment(genDecl, typeSpec)
			capAnnotation := parseCapabilityAnnotation(rawDoc)
			if capAnnotation == nil {
				continue
			}

			// Extract source file base name (e.g., "websocket_callback" from "websocket_callback.go")
			baseName := filepath.Base(path)
			sourceFile := strings.TrimSuffix(baseName, ".go")

			capability := Capability{
				Name:       capAnnotation["name"],
				Interface:  typeSpec.Name.Name,
				Required:   capAnnotation["required"] == "true",
				Doc:        cleanDoc(docText),
				SourceFile: sourceFile,
			}

			// Parse methods and collect referenced types
			referencedTypes := make(map[string]bool)
			for _, method := range interfaceType.Methods.List {
				if len(method.Names) == 0 {
					continue // Embedded interface
				}

				funcType, ok := method.Type.(*ast.FuncType)
				if !ok {
					continue
				}

				// Check for //nd:export annotation
				methodDocText, methodRawDoc := getMethodDocComment(method)
				exportAnnotation := parseExportAnnotation(methodRawDoc)
				if exportAnnotation == nil {
					continue
				}

				export, err := parseExport(method.Names[0].Name, funcType, exportAnnotation, cleanDoc(methodDocText))
				if err != nil {
					return nil, fmt.Errorf("parsing export %s.%s: %w", typeSpec.Name.Name, method.Names[0].Name, err)
				}
				capability.Methods = append(capability.Methods, export)

				// Collect referenced types from input and output
				collectReferencedTypes(export.Input.Type, referencedTypes)
				collectReferencedTypes(export.Output.Type, referencedTypes)
			}

			// Recursively collect all struct dependencies
			collectAllStructDependencies(referencedTypes, structMap)

			// Sort type names for stable output order
			sortedTypeNames := slices.Sorted(maps.Keys(referencedTypes))

			// Attach referenced structs to the capability
			for _, typeName := range sortedTypeNames {
				if s, exists := structMap[typeName]; exists {
					capability.Structs = append(capability.Structs, s)
				}
			}

			// Attach referenced type aliases
			for _, typeName := range sortedTypeNames {
				if a, exists := aliasMap[typeName]; exists {
					capability.TypeAliases = append(capability.TypeAliases, a)
				}
			}

			// Also attach type aliases prefixed with interface name (e.g., ScrobblerError for Scrobbler interface)
			// This supports error types that are not directly referenced in method signatures
			interfaceName := typeSpec.Name.Name
			for _, typeName := range slices.Sorted(maps.Keys(aliasMap)) {
				a := aliasMap[typeName]
				if strings.HasPrefix(typeName, interfaceName) && !referencedTypes[typeName] {
					capability.TypeAliases = append(capability.TypeAliases, a)
					referencedTypes[typeName] = true // Mark as referenced for const lookup
				}
			}

			// Attach const groups that match referenced type aliases
			for _, group := range allConstGroups {
				if group.Type == "" {
					continue
				}
				if referencedTypes[group.Type] {
					capability.Consts = append(capability.Consts, group)
				}
			}

			if len(capability.Methods) > 0 {
				capabilities = append(capabilities, capability)
			}
		}
	}

	return capabilities, nil
}

// collectAllStructDependencies recursively collects all struct types referenced by other structs.
func collectAllStructDependencies(referencedTypes map[string]bool, structMap map[string]StructDef) {
	// Keep iterating until no new types are added
	for {
		newTypes := make(map[string]bool)
		for typeName := range referencedTypes {
			if s, exists := structMap[typeName]; exists {
				for _, field := range s.Fields {
					collectReferencedTypes(field.Type, newTypes)
				}
			}
		}
		// Check if any new types were found
		foundNew := false
		for t := range newTypes {
			if !referencedTypes[t] {
				referencedTypes[t] = true
				foundNew = true
			}
		}
		if !foundNew {
			break
		}
	}
}

// parseExport parses an export method signature into an Export struct.
func parseExport(name string, funcType *ast.FuncType, annotation map[string]string, doc string) (Export, error) {
	export := Export{
		Name:       name,
		ExportName: annotation["name"],
		Doc:        doc,
	}

	// Capability exports have exactly one input parameter (the struct type)
	if funcType.Params != nil && len(funcType.Params.List) == 1 {
		field := funcType.Params.List[0]
		typeName := typeToString(field.Type)
		paramName := "input"
		if len(field.Names) > 0 {
			paramName = field.Names[0].Name
		}
		export.Input = NewParam(paramName, typeName)
	}

	// Capability exports return (OutputType, error)
	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			typeName := typeToString(field.Type)
			if typeName == "error" {
				continue // Skip error return
			}
			paramName := "output"
			if len(field.Names) > 0 {
				paramName = field.Names[0].Name
			}
			export.Output = NewParam(paramName, typeName)
			break // Only take the first non-error return
		}
	}

	return export, nil
}

// parseFile parses a single Go source file and extracts host services.
func parseFile(fset *token.FileSet, path string) ([]Service, error) {
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// First pass: collect all struct definitions in the file
	allStructs := parseStructs(f)
	structMap := make(map[string]StructDef)
	for _, s := range allStructs {
		structMap[s.Name] = s
	}

	var services []Service

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			// Check for //nd:hostservice annotation in doc comment
			docText, rawDoc := getDocComment(genDecl, typeSpec)
			svcAnnotation := parseHostServiceAnnotation(rawDoc)
			if svcAnnotation == nil {
				continue
			}

			service := Service{
				Name:       svcAnnotation["name"],
				Permission: svcAnnotation["permission"],
				Interface:  typeSpec.Name.Name,
				Doc:        cleanDoc(docText),
			}

			// Parse methods and collect referenced types
			referencedTypes := make(map[string]bool)
			for _, method := range interfaceType.Methods.List {
				if len(method.Names) == 0 {
					continue // Embedded interface
				}

				funcType, ok := method.Type.(*ast.FuncType)
				if !ok {
					continue
				}

				// Check for //nd:hostfunc annotation
				methodDocText, methodRawDoc := getMethodDocComment(method)
				methodAnnotation := parseHostFuncAnnotation(methodRawDoc)
				if methodAnnotation == nil {
					continue
				}

				m, err := parseMethod(method.Names[0].Name, funcType, methodAnnotation, cleanDoc(methodDocText))
				if err != nil {
					return nil, fmt.Errorf("parsing method %s.%s: %w", typeSpec.Name.Name, method.Names[0].Name, err)
				}
				service.Methods = append(service.Methods, m)

				// Collect referenced types from params and returns
				for _, p := range m.Params {
					collectReferencedTypes(p.Type, referencedTypes)
				}
				for _, r := range m.Returns {
					collectReferencedTypes(r.Type, referencedTypes)
				}
			}

			// Attach referenced structs to the service (sorted for stable output)
			for _, typeName := range slices.Sorted(maps.Keys(referencedTypes)) {
				if s, exists := structMap[typeName]; exists {
					service.Structs = append(service.Structs, s)
				}
			}

			if len(service.Methods) > 0 {
				services = append(services, service)
			}
		}
	}

	return services, nil
}

// parseStructs extracts all struct type definitions from a parsed Go file.
func parseStructs(f *ast.File) []StructDef {
	var structs []StructDef

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			docText, _ := getDocComment(genDecl, typeSpec)
			s := StructDef{
				Name: typeSpec.Name.Name,
				Doc:  cleanDoc(docText),
			}

			// Parse struct fields
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue // Embedded field
				}

				fieldDef := parseStructField(field)
				s.Fields = append(s.Fields, fieldDef...)
			}

			structs = append(structs, s)
		}
	}

	return structs
}

// parseTypeAliases extracts all type alias definitions from a parsed Go file.
// Type aliases are non-struct type declarations like: type MyType string
func parseTypeAliases(f *ast.File) []TypeAlias {
	var aliases []TypeAlias

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Skip struct and interface types
			if _, isStruct := typeSpec.Type.(*ast.StructType); isStruct {
				continue
			}
			if _, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
				continue
			}

			docText, _ := getDocComment(genDecl, typeSpec)
			aliases = append(aliases, TypeAlias{
				Name: typeSpec.Name.Name,
				Type: typeToString(typeSpec.Type),
				Doc:  cleanDoc(docText),
			})
		}
	}

	return aliases
}

// parseConstGroups extracts const groups from a parsed Go file.
func parseConstGroups(f *ast.File) []ConstGroup {
	var groups []ConstGroup

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		group := ConstGroup{}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Get type if specified
			if valueSpec.Type != nil && group.Type == "" {
				group.Type = typeToString(valueSpec.Type)
			}

			// Extract values
			for i, name := range valueSpec.Names {
				def := ConstDef{
					Name: name.Name,
				}
				// Get value if present
				if i < len(valueSpec.Values) {
					def.Value = exprToString(valueSpec.Values[i])
				}
				// Get doc comment
				if valueSpec.Doc != nil {
					def.Doc = cleanDoc(valueSpec.Doc.Text())
				} else if valueSpec.Comment != nil {
					def.Doc = cleanDoc(valueSpec.Comment.Text())
				}
				group.Values = append(group.Values, def)
			}
		}

		if len(group.Values) > 0 {
			groups = append(groups, group)
		}
	}

	return groups
}

// exprToString converts an AST expression to a Go source string.
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	default:
		return ""
	}
}

// parseStructField parses a struct field and returns FieldDef for each name.
func parseStructField(field *ast.Field) []FieldDef {
	var fields []FieldDef
	typeName := typeToString(field.Type)

	// Parse struct tag for JSON field name and omitempty
	jsonTag := ""
	omitEmpty := false
	if field.Tag != nil {
		tag := field.Tag.Value
		// Remove backticks
		tag = strings.Trim(tag, "`")
		// Parse json tag
		jsonTag, omitEmpty = parseJSONTag(tag)
	}

	// Get doc comment
	var doc string
	if field.Doc != nil {
		doc = cleanDoc(field.Doc.Text())
	}

	for _, name := range field.Names {
		fieldJSONTag := jsonTag
		if fieldJSONTag == "" {
			// Default to field name with camelCase
			fieldJSONTag = toJSONName(name.Name)
		}
		fields = append(fields, FieldDef{
			Name:      name.Name,
			Type:      typeName,
			JSONTag:   fieldJSONTag,
			OmitEmpty: omitEmpty,
			Doc:       doc,
		})
	}

	return fields
}

// parseJSONTag extracts the json field name and omitempty flag from a struct tag.
func parseJSONTag(tag string) (name string, omitEmpty bool) {
	// Find json:"..." in the tag
	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, `json:"`) {
			value := strings.TrimPrefix(part, `json:"`)
			value = strings.TrimSuffix(value, `"`)
			parts := strings.Split(value, ",")
			if len(parts) > 0 && parts[0] != "-" {
				name = parts[0]
			}
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					omitEmpty = true
				}
			}
			return
		}
	}
	return "", false
}

// collectReferencedTypes extracts custom type names from a Go type string.
// It handles pointers, slices, and maps, collecting base type names.
func collectReferencedTypes(goType string, refs map[string]bool) {
	// Strip pointer
	if strings.HasPrefix(goType, "*") {
		collectReferencedTypes(goType[1:], refs)
		return
	}
	// Strip slice
	if strings.HasPrefix(goType, "[]") {
		if goType != "[]byte" {
			collectReferencedTypes(goType[2:], refs)
		}
		return
	}
	// Handle map
	if strings.HasPrefix(goType, "map[") {
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
		collectReferencedTypes(keyType, refs)
		collectReferencedTypes(valueType, refs)
		return
	}

	// Check if it's a custom type (starts with uppercase, not a builtin)
	if len(goType) > 0 && goType[0] >= 'A' && goType[0] <= 'Z' {
		switch goType {
		case "String", "Bool", "Int", "Int32", "Int64", "Float32", "Float64":
			// Not custom types (just capitalized for some reason)
		default:
			refs[goType] = true
		}
	}
}

// toJSONName is imported from types.go via the same package

// getDocComment extracts the doc comment for a type spec.
// Returns both the readable doc text and the raw comment text (which includes pragma-style comments).
func getDocComment(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) (docText, rawText string) {
	var docGroup *ast.CommentGroup
	// First check the TypeSpec's own doc (when multiple types in one block)
	if typeSpec.Doc != nil {
		docGroup = typeSpec.Doc
	} else if genDecl.Doc != nil {
		// Fall back to GenDecl doc (single type declaration)
		docGroup = genDecl.Doc
	}
	if docGroup == nil {
		return "", ""
	}
	return docGroup.Text(), commentGroupRaw(docGroup)
}

// commentGroupRaw returns all comment text including pragma-style comments (//nd:...).
// Go's ast.CommentGroup.Text() strips comments without a space after //, so we need this.
func commentGroupRaw(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var lines []string
	for _, c := range cg.List {
		lines = append(lines, c.Text)
	}
	return strings.Join(lines, "\n")
}

// getMethodDocComment extracts the doc comment for a method.
func getMethodDocComment(field *ast.Field) (docText, rawText string) {
	if field.Doc == nil {
		return "", ""
	}
	return field.Doc.Text(), commentGroupRaw(field.Doc)
}

// parseHostServiceAnnotation extracts //nd:hostservice annotation parameters.
func parseHostServiceAnnotation(doc string) map[string]string {
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSpace(line)
		match := hostServicePattern.FindStringSubmatch(line)
		if match != nil {
			return parseKeyValuePairs(match[1])
		}
	}
	return nil
}

// parseHostFuncAnnotation extracts //nd:hostfunc annotation parameters.
func parseHostFuncAnnotation(doc string) map[string]string {
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSpace(line)
		match := hostFuncPattern.FindStringSubmatch(line)
		if match != nil {
			params := parseKeyValuePairs(match[1])
			if params == nil {
				params = make(map[string]string)
			}
			return params
		}
	}
	return nil
}

// parseCapabilityAnnotation extracts //nd:capability annotation parameters.
func parseCapabilityAnnotation(doc string) map[string]string {
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSpace(line)
		match := capabilityPattern.FindStringSubmatch(line)
		if match != nil {
			return parseKeyValuePairs(match[1])
		}
	}
	return nil
}

// parseExportAnnotation extracts //nd:export annotation parameters.
func parseExportAnnotation(doc string) map[string]string {
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSpace(line)
		match := exportPattern.FindStringSubmatch(line)
		if match != nil {
			return parseKeyValuePairs(match[1])
		}
	}
	return nil
}

// parseKeyValuePairs extracts key=value pairs from annotation text.
func parseKeyValuePairs(text string) map[string]string {
	matches := keyValuePattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, m := range matches {
		result[m[1]] = m[2]
	}
	return result
}

// parseMethod parses a method signature into a Method struct.
func parseMethod(name string, funcType *ast.FuncType, annotation map[string]string, doc string) (Method, error) {
	m := Method{
		Name:       name,
		ExportName: annotation["name"],
		Doc:        doc,
	}

	// Parse parameters (skip context.Context)
	if funcType.Params != nil {
		for _, field := range funcType.Params.List {
			typeName := typeToString(field.Type)
			if typeName == "context.Context" {
				continue // Skip context parameter
			}

			for _, name := range field.Names {
				m.Params = append(m.Params, NewParam(name.Name, typeName))
			}
		}
	}

	// Parse return values
	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			typeName := typeToString(field.Type)
			if typeName == "error" {
				m.HasError = true
				continue // Track error but don't include in Returns
			}

			// Handle anonymous returns
			if len(field.Names) == 0 {
				// Generate a name based on position
				m.Returns = append(m.Returns, NewParam("result", typeName))
			} else {
				for _, name := range field.Names {
					m.Returns = append(m.Returns, NewParam(name.Name, typeName))
				}
			}
		}
	}

	return m, nil
}

// typeToString converts an AST type expression to a string.
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeToString(t.Elt)
		}
		return fmt.Sprintf("[%s]%s", typeToString(t.Len), typeToString(t.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", typeToString(t.Key), typeToString(t.Value))
	case *ast.BasicLit:
		return t.Value
	case *ast.InterfaceType:
		// Empty interface (interface{} or any)
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "any"
		}
		// Non-empty interfaces can't be easily represented
		return "any"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// cleanDoc removes annotation lines from documentation.
func cleanDoc(doc string) string {
	var lines []string
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//nd:") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
