package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// PDKSymbols contains all exported symbols parsed from extism/go-pdk.
type PDKSymbols struct {
	Types     []PDKType
	Consts    []PDKConst
	Functions []PDKFunc
}

// PDKType represents an exported type from extism/go-pdk.
type PDKType struct {
	Name       string
	Underlying string     // The underlying type (e.g., "int" for LogLevel)
	IsAlias    bool       // True if it's a type alias (type X = Y)
	Doc        string     // Documentation comment
	Methods    []PDKFunc  // Methods on this type
	Fields     []PDKField // Struct fields (if it's a struct type)
}

// PDKField represents a struct field.
type PDKField struct {
	Name string
	Type string
	Tag  string // Struct tag (e.g., `json:"name"`)
}

// PDKConst represents an exported constant from extism/go-pdk.
type PDKConst struct {
	Name  string
	Type  string // The type name (may be empty for untyped consts)
	Value string // The value expression
	Doc   string
}

// PDKFunc represents an exported function from extism/go-pdk.
type PDKFunc struct {
	Name       string
	Doc        string
	Receiver   string // Empty for package-level functions
	Params     []PDKParam
	Returns    []PDKReturn
	IsVariadic bool
}

// PDKParam represents a function parameter.
type PDKParam struct {
	Name string
	Type string
}

// PDKReturn represents a function return value.
type PDKReturn struct {
	Name string // May be empty for unnamed returns
	Type string
}

// ParseExtismPDK parses the extism/go-pdk package and extracts all exported symbols.
func ParseExtismPDK() (*PDKSymbols, error) {
	// Load both packages with syntax trees in one call
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg,
		"github.com/extism/go-pdk",
		"github.com/extism/go-pdk/internal/memory",
	)
	if err != nil {
		return nil, fmt.Errorf("loading extism/go-pdk: %w", err)
	}

	// Find both packages
	var pdkPkg, memoryPkg *packages.Package
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, fmt.Errorf("loading %s: %v", pkg.PkgPath, pkg.Errors[0])
		}
		switch pkg.Name {
		case "pdk":
			pdkPkg = pkg
		case "memory":
			memoryPkg = pkg
		}
	}
	if pdkPkg == nil {
		return nil, fmt.Errorf("package github.com/extism/go-pdk not found")
	}
	if memoryPkg == nil {
		return nil, fmt.Errorf("package github.com/extism/go-pdk/internal/memory not found")
	}

	symbols := &PDKSymbols{}
	seenTypes := make(map[string]bool)

	// Extract Memory type from internal/memory package first
	extractMemorySymbols(memoryPkg.Syntax, symbols, seenTypes)

	// First pass: collect types from pdk package (skip if already found in internal packages)
	for _, file := range pdkPkg.Syntax {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if !typeSpec.Name.IsExported() {
							continue
						}
						// Skip if we already have this type (from internal packages)
						if seenTypes[typeSpec.Name.Name] {
							continue
						}
						seenTypes[typeSpec.Name.Name] = true
						pdkType := extractType(typeSpec, genDecl.Doc)
						symbols.Types = append(symbols.Types, pdkType)
					}
				}
			}
		}
	}

	// Build typeMap from the final slice (after all types are added)
	typeMap := make(map[string]*PDKType)
	for i := range symbols.Types {
		typeMap[symbols.Types[i].Name] = &symbols.Types[i]
	}

	// Second pass: collect functions and methods from pdk package
	for _, file := range pdkPkg.Syntax {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok == token.CONST {
					consts := extractConsts(d)
					symbols.Consts = append(symbols.Consts, consts...)
				}
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				fn := extractFunc(d)
				if fn.Receiver != "" {
					// It's a method, associate with type
					typeName := fn.Receiver
					if strings.HasPrefix(typeName, "*") {
						typeName = typeName[1:]
					}
					if t, ok := typeMap[typeName]; ok {
						t.Methods = append(t.Methods, fn)
					}
				} else {
					symbols.Functions = append(symbols.Functions, fn)
				}
			}
		}
	}

	// Sort for consistent output
	sort.Slice(symbols.Types, func(i, j int) bool {
		return symbols.Types[i].Name < symbols.Types[j].Name
	})
	sort.Slice(symbols.Consts, func(i, j int) bool {
		return symbols.Consts[i].Name < symbols.Consts[j].Name
	})
	sort.Slice(symbols.Functions, func(i, j int) bool {
		return symbols.Functions[i].Name < symbols.Functions[j].Name
	})

	return symbols, nil
}

func extractType(spec *ast.TypeSpec, doc *ast.CommentGroup) PDKType {
	t := PDKType{
		Name: spec.Name.Name,
		Doc:  extractDoc(doc),
	}

	// Check if it's an alias (type X = Y)
	t.IsAlias = spec.Assign.IsValid()

	// Extract underlying type
	t.Underlying = typeString(spec.Type)

	// Extract struct fields if it's a struct type
	if structType, ok := spec.Type.(*ast.StructType); ok {
		t.Fields = extractStructFields(structType)
	}

	return t
}

func extractStructFields(st *ast.StructType) []PDKField {
	var fields []PDKField
	if st.Fields == nil {
		return fields
	}

	for _, field := range st.Fields.List {
		fieldType := typeString(field.Type)
		tag := ""
		if field.Tag != nil {
			tag = field.Tag.Value
		}

		if len(field.Names) == 0 {
			// Embedded field
			fields = append(fields, PDKField{
				Name: fieldType, // Use type name as field name for embedded
				Type: fieldType,
				Tag:  tag,
			})
		} else {
			for _, name := range field.Names {
				// Skip unexported fields
				if !name.IsExported() {
					continue
				}
				fields = append(fields, PDKField{
					Name: name.Name,
					Type: fieldType,
					Tag:  tag,
				})
			}
		}
	}
	return fields
}

func extractConsts(decl *ast.GenDecl) []PDKConst {
	var consts []PDKConst
	var currentType string // For iota-style const blocks

	for i, spec := range decl.Specs {
		valSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// Update type if specified
		if valSpec.Type != nil {
			currentType = typeString(valSpec.Type)
		}

		for j, name := range valSpec.Names {
			if !name.IsExported() {
				continue
			}

			c := PDKConst{
				Name: name.Name,
				Type: currentType,
			}

			// Extract value
			if j < len(valSpec.Values) {
				c.Value = exprString(valSpec.Values[j])
			} else if i == 0 && j == 0 {
				// First const with no value - likely iota
				c.Value = "iota"
			}

			// Extract doc
			if valSpec.Doc != nil {
				c.Doc = extractDoc(valSpec.Doc)
			} else if i == 0 && decl.Doc != nil {
				c.Doc = extractDoc(decl.Doc)
			}

			consts = append(consts, c)
		}
	}

	return consts
}

func extractFunc(decl *ast.FuncDecl) PDKFunc {
	fn := PDKFunc{
		Name: decl.Name.Name,
		Doc:  extractDoc(decl.Doc),
	}

	// Extract receiver
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		fn.Receiver = typeString(decl.Recv.List[0].Type)
	}

	// Extract parameters
	if decl.Type.Params != nil {
		for _, field := range decl.Type.Params.List {
			paramType := typeString(field.Type)

			// Check for variadic
			if _, ok := field.Type.(*ast.Ellipsis); ok {
				fn.IsVariadic = true
			}

			if len(field.Names) == 0 {
				// Unnamed parameter
				fn.Params = append(fn.Params, PDKParam{Type: paramType})
			} else {
				for _, name := range field.Names {
					fn.Params = append(fn.Params, PDKParam{
						Name: name.Name,
						Type: paramType,
					})
				}
			}
		}
	}

	// Extract returns
	if decl.Type.Results != nil {
		for _, field := range decl.Type.Results.List {
			retType := typeString(field.Type)

			if len(field.Names) == 0 {
				// Unnamed return
				fn.Returns = append(fn.Returns, PDKReturn{Type: retType})
			} else {
				for _, name := range field.Names {
					fn.Returns = append(fn.Returns, PDKReturn{
						Name: name.Name,
						Type: retType,
					})
				}
			}
		}
	}

	return fn
}

func extractDoc(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	return strings.TrimSpace(doc.Text())
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(t.Elt)
		}
		return fmt.Sprintf("[%s]%s", exprString(t.Len), typeString(t.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", typeString(t.Key), typeString(t.Value))
	case *ast.InterfaceType:
		return "any" // Simplified
	case *ast.Ellipsis:
		return "..." + typeString(t.Elt)
	case *ast.StructType:
		return "struct{}" // Simplified for anonymous structs
	case *ast.FuncType:
		return "func()" // Simplified
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.BasicLit:
		return e.Value
	case *ast.BinaryExpr:
		return exprString(e.X) + " " + e.Op.String() + " " + exprString(e.Y)
	case *ast.UnaryExpr:
		return e.Op.String() + exprString(e.X)
	case *ast.CallExpr:
		return typeString(e.Fun) + "(...)"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// extractMemorySymbols extracts the Memory type and its methods from already-parsed syntax trees.
// This is needed because Memory is defined in internal/memory but re-exported by the pdk package.
func extractMemorySymbols(files []*ast.File, symbols *PDKSymbols, seenTypes map[string]bool) {
	// Collect the Memory type
	for _, file := range files {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						// Only interested in Memory type
						if typeSpec.Name.Name == "Memory" {
							pdkType := extractType(typeSpec, genDecl.Doc)
							symbols.Types = append(symbols.Types, pdkType)
							seenTypes["Memory"] = true
						}
					}
				}
			}
		}
	}

	// Build local type map for method association
	localTypeMap := make(map[string]*PDKType)
	for i := range symbols.Types {
		localTypeMap[symbols.Types[i].Name] = &symbols.Types[i]
	}

	// Collect methods for Memory
	for _, file := range files {
		for _, decl := range file.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				if !funcDecl.Name.IsExported() {
					continue
				}
				fn := extractFunc(funcDecl)
				if fn.Receiver != "" {
					typeName := fn.Receiver
					if strings.HasPrefix(typeName, "*") {
						typeName = typeName[1:]
					}
					if typeName == "Memory" {
						if t, ok := localTypeMap["Memory"]; ok {
							t.Methods = append(t.Methods, fn)
						}
					}
				}
			}
		}
	}
}
