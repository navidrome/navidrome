package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Annotation patterns
var (
	// //nd:hostservice name=ServiceName permission=key
	hostServicePattern = regexp.MustCompile(`//nd:hostservice\s+(.*)`)
	// //nd:hostfunc [name=CustomName]
	hostFuncPattern = regexp.MustCompile(`//nd:hostfunc(?:\s+(.*))?`)
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

// parseFile parses a single Go source file and extracts host services.
func parseFile(fset *token.FileSet, path string) ([]Service, error) {
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
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

			// Parse methods
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
			}

			if len(service.Methods) > 0 {
				services = append(services, service)
			}
		}
	}

	return services, nil
}

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
