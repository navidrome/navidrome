// Package ctxvariant reports calls to F when a drop-in sibling FContext or FWithContext takes a leading
// context.Context, and calls to stdlib APIs whose ctx-aware replacement is named differently.
package ctxvariant

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

var Analyzer = &analysis.Analyzer{
	Name:     "ctxvariant",
	Doc:      "reports calls to F when F's package or receiver also has FContext or FWithContext taking a leading context.Context, and calls to known ctx-less stdlib APIs",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var suffixes = []string{"Context", "WithContext"}

// irregular maps stdlib calls whose ctx-aware replacement does not follow the FContext naming.
var irregular = map[string]string{
	"(*database/sql.DB).Begin":    "(*database/sql.DB).BeginTx",
	"net/http.Get":                "net/http.NewRequestWithContext with (*net/http.Client).Do",
	"net/http.Head":               "net/http.NewRequestWithContext with (*net/http.Client).Do",
	"net/http.Post":               "net/http.NewRequestWithContext with (*net/http.Client).Do",
	"net/http.PostForm":           "net/http.NewRequestWithContext with (*net/http.Client).Do",
	"(*net/http.Client).Get":      "net/http.NewRequestWithContext with (*net/http.Client).Do",
	"(*net/http.Client).Head":     "net/http.NewRequestWithContext with (*net/http.Client).Do",
	"(*net/http.Client).Post":     "net/http.NewRequestWithContext with (*net/http.Client).Do",
	"(*net/http.Client).PostForm": "net/http.NewRequestWithContext with (*net/http.Client).Do",
	"net.Dial":                    "(*net.Dialer).DialContext",
	"net.DialTimeout":             "(*net.Dialer).DialContext",
	"net.Listen":                  "(*net.ListenConfig).Listen",
	"net.ListenPacket":            "(*net.ListenConfig).ListenPacket",
	"net.LookupAddr":              "(*net.Resolver).LookupAddr",
	"net.LookupCNAME":             "(*net.Resolver).LookupCNAME",
	"net.LookupHost":              "(*net.Resolver).LookupHost",
	"net.LookupIP":                "(*net.Resolver).LookupIP",
	"net.LookupMX":                "(*net.Resolver).LookupMX",
	"net.LookupNS":                "(*net.Resolver).LookupNS",
	"net.LookupPort":              "(*net.Resolver).LookupPort",
	"net.LookupSRV":               "(*net.Resolver).LookupSRV",
	"net.LookupTXT":               "(*net.Resolver).LookupTXT",
	"crypto/tls.Dial":             "(*crypto/tls.Dialer).DialContext",
	"crypto/tls.DialWithDialer":   "(*crypto/tls.Dialer).DialContext",
}

type variant struct {
	name string
	decl *types.Func // nil for irregular replacements
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	variants := map[*types.Func]variant{}
	var enclosing *types.Func

	filter := []ast.Node{(*ast.FuncDecl)(nil), (*ast.CallExpr)(nil)}
	insp.Nodes(filter, func(n ast.Node, push bool) bool {
		if decl, ok := n.(*ast.FuncDecl); ok {
			enclosing = nil
			if push {
				enclosing, _ = pass.TypesInfo.Defs[decl.Name].(*types.Func)
			}
			return true
		}
		if !push {
			return true
		}
		call := n.(*ast.CallExpr)
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if !ok {
			return true
		}
		fn = fn.Origin()
		v, cached := variants[fn]
		if !cached {
			v = findVariant(fn)
			variants[fn] = v
		}
		if v.name == "" || enclosing == fn || (v.decl != nil && enclosing == v.decl) {
			return true
		}
		pass.Reportf(call.Lparen, "use %s and pass the caller's ctx", v.name)
		return true
	})
	return nil, nil
}

func findVariant(fn *types.Func) variant {
	if fn.Pkg() == nil {
		return variant{}
	}
	if name, ok := irregular[fn.FullName()]; ok {
		return variant{name: name}
	}
	recv := fn.Signature().Recv()
	for _, suffix := range suffixes {
		name := fn.Name() + suffix
		var obj types.Object
		if recv == nil {
			obj = fn.Pkg().Scope().Lookup(name)
		} else {
			var index []int
			obj, index, _ = types.LookupFieldOrMethod(recv.Type(), true, fn.Pkg(), name)
			// A sibling promoted from an embedded field would skip fn's own override.
			if len(index) > 1 {
				continue
			}
		}
		if sib, ok := obj.(*types.Func); ok && isDropIn(fn.Signature(), sib.Signature()) {
			sib = sib.Origin()
			return variant{name: sib.FullName(), decl: sib}
		}
	}
	return variant{}
}

// isDropIn reports whether sib's parameters are ctx followed by exactly f's parameters, with the same results.
func isDropIn(f, sib *types.Signature) bool {
	if f.TypeParams().Len() != sib.TypeParams().Len() || sib.Params().Len() != f.Params().Len()+1 ||
		sib.Variadic() != f.Variadic() || !isContext(sib.Params().At(0).Type()) {
		return false
	}
	if n := f.TypeParams().Len(); n > 0 {
		// Generic siblings declare their own type parameters; rename them to f's before comparing.
		args := make([]types.Type, n)
		for i := range n {
			args[i] = f.TypeParams().At(i)
		}
		inst, err := types.Instantiate(nil, sib, args, true)
		if err != nil {
			return false
		}
		sib = inst.(*types.Signature)
	}
	for i := range f.Params().Len() {
		if !types.Identical(f.Params().At(i).Type(), sib.Params().At(i+1).Type()) {
			return false
		}
	}
	return types.Identical(f.Results(), sib.Results())
}

func isContext(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
}
