package subset

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/eugenioenko/goalchemy/internal/catalog"
	"github.com/eugenioenko/goalchemy/internal/ir"
)

const fmtPackage = catalog.StdModule + "fmt"

func calleeIdent(fun ast.Expr) *ast.Ident {
	switch f := ast.Unparen(fun).(type) {
	case *ast.SelectorExpr:
		return f.Sel
	case *ast.Ident:
		return f
	}
	return nil
}

// libraryCall checks standard library calls that Go implements with
// reflection: errors.As targets and fmt operands must have static types the
// compiler and std/fmt can handle without it.
func (c *checker) libraryCall(n *ast.CallExpr, fun ast.Expr) {
	id := calleeIdent(fun)
	if id == nil {
		return
	}
	fn, ok := c.info.Uses[id].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return
	}
	switch {
	case catalog.IsErrorsAs(fn):
		c.called[id] = true
		c.errorsAs(n, fn)
	case fn.Pkg().Path() == fmtPackage && fn.Signature().Variadic() && !n.Ellipsis.IsValid():
		params := fn.Signature().Params()
		for _, a := range n.Args[min(params.Len()-1, len(n.Args)):] {
			c.fmtOperand(a)
		}
	}
}

func (c *checker) errorsAs(n *ast.CallExpr, fn *types.Func) {
	if c.pkg.PkgPath == catalog.ErrorsPackage {
		return
	}
	if fn.Pkg().Path() != catalog.ErrorsPackage {
		c.unsupported(n, "GCS006", "lib/errors.As", "Import github.com/eugenioenko/goalchemy/std/errors and call errors.As.")
		return
	}
	if len(n.Args) != 2 {
		c.unsupported(n, "GCS006", "errors.As with a multi-value argument", "Pass the error and the target pointer as separate arguments.")
		return
	}
	t := c.info.TypeOf(n.Args[1])
	if t == nil {
		return
	}
	p, ok := t.Underlying().(*types.Pointer)
	if !ok {
		c.report(n.Args[1], "GCS006", "errors.As target", "errors.As target of static type "+typeName(t)+" is not supported",
			"Pass a pointer such as &target, where target is an interface type or implements error.")
		return
	}
	if !types.IsInterface(p.Elem()) && !types.Implements(p.Elem(), errorInterface) {
		remedy := "Point target at an interface type or at a type implementing error."
		if types.Implements(types.NewPointer(p.Elem()), errorInterface) {
			remedy = "Use a pointer to the pointer type, such as var target *T; errors.As(err, &target)."
		}
		c.report(n.Args[1], "GCS006", "errors.As target", "errors.As target *"+typeName(p.Elem())+" must point to an interface type or a type implementing error", remedy)
	}
}

func (c *checker) deferredIntrinsic(call *ast.CallExpr, stmt string) {
	if id := calleeIdent(call.Fun); id != nil && catalog.IsErrorsAs(c.info.Uses[id]) {
		c.unsupported(call, "GCS006", stmt+" errors.As", "Call errors.As directly and use its result.")
	}
}

var errorInterface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

var stringerInterface = types.NewInterfaceType([]*types.Func{
	types.NewFunc(token.NoPos, nil, "String", types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false)),
}, nil).Complete()

// fmtOperand rejects operands std/fmt cannot format without reflection:
// values other than basic types, []byte, interfaces, and types with an
// Error, String, GoString or Format method.
func (c *checker) fmtOperand(a ast.Expr) {
	tv, ok := c.info.Types[a]
	if !ok || tv.Type == nil || tv.IsNil() {
		return
	}
	t := tv.Type
	if b, ok := t.(*types.Basic); ok && b.Info()&types.IsUntyped != 0 {
		return
	}
	if types.IsInterface(t) || types.Implements(t, errorInterface) || types.Implements(t, stringerInterface) || hasMethod(t, "Format") || hasMethod(t, "GoString") {
		return
	}
	switch u := types.Unalias(t).(type) {
	case *types.Basic:
		if u.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsString) != 0 {
			return
		}
	case *types.Slice:
		if b, ok := types.Unalias(u.Elem()).(*types.Basic); ok && b.Kind() == types.Uint8 {
			return
		}
	}
	c.report(a, "GCS006", "fmt operand of type "+typeName(t),
		"fmt operand of type "+typeName(t)+" is not supported: std/fmt formats basic types, []byte, interfaces, and values with an Error or String method without reflection",
		"Convert named basic types to their basic type, such as int(x), or give the type a String() string method.")
}

func hasMethod(t types.Type, name string) bool {
	obj, _, _ := types.LookupFieldOrMethod(t, false, nil, name)
	fn, ok := obj.(*types.Func)
	return ok && ir.MethodID(fn) == fn.Id()
}

func typeName(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}
