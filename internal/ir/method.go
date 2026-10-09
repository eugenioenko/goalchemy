package ir

import (
	"go/types"
	"hash/fnv"
	"strconv"
)

const fmtPackage = "github.com/eugenioenko/goalchemy/std/fmt"

// MethodID is the identity targets use to match methods in dispatch tables
// and interface checks: Go's Id, plus a signature fingerprint when a method
// reuses a well-known name with a different signature. Targets compare IDs
// as strings, so without the fingerprint a type declaring Unwrap() []error
// would satisfy interface{ Unwrap() error }.
func MethodID(fn *types.Func) string {
	sig := fn.Signature()
	if canonical, known := wellKnown(fn.Name(), sig); !known || canonical {
		return fn.Id()
	}
	h := fnv.New32a()
	h.Write([]byte(types.TypeString(types.NewSignatureType(nil, nil, nil, sig.Params(), sig.Results(), sig.Variadic()),
		func(p *types.Package) string { return p.Path() })))
	return fn.Id() + "#" + strconv.FormatUint(uint64(h.Sum32()), 16)
}

func wellKnown(name string, sig *types.Signature) (canonical, known bool) {
	params, results := sig.Params(), sig.Results()
	switch name {
	case "Error", "String", "GoString":
		return params.Len() == 0 && results.Len() == 1 && isBasic(results.At(0).Type(), types.String), true
	case "Unwrap":
		return params.Len() == 0 && results.Len() == 1 && isError(results.At(0).Type()), true
	case "Is":
		return !sig.Variadic() && params.Len() == 1 && isError(params.At(0).Type()) &&
			results.Len() == 1 && isBasic(results.At(0).Type(), types.Bool), true
	case "As":
		if params.Len() != 1 || sig.Variadic() || results.Len() != 1 || !isBasic(results.At(0).Type(), types.Bool) {
			return false, true
		}
		i, ok := params.At(0).Type().(*types.Interface)
		if a, isAlias := params.At(0).Type().(*types.Alias); isAlias {
			i, ok = types.Unalias(a).(*types.Interface)
		}
		return ok && i.Empty(), true
	case "Format":
		if params.Len() != 2 || sig.Variadic() || results.Len() != 0 || !isBasic(params.At(1).Type(), types.Int32) {
			return false, true
		}
		n, ok := types.Unalias(params.At(0).Type()).(*types.Named)
		return ok && n.Obj().Name() == "State" && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == fmtPackage, true
	}
	return false, false
}

func isBasic(t types.Type, kind types.BasicKind) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == kind
}

func isError(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}
