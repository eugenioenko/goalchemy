package ir

import (
	"go/token"
	"go/types"
	"hash/fnv"
	"strconv"

	"golang.org/x/tools/go/types/typeutil"
)

func signature(params []types.Type, results ...types.Type) *types.Signature {
	tuple := func(ts []types.Type) *types.Tuple {
		var vs []*types.Var
		for _, t := range ts {
			vs = append(vs, types.NewVar(token.NoPos, nil, "", t))
		}
		return types.NewTuple(vs...)
	}
	return types.NewSignatureType(nil, nil, nil, tuple(params), tuple(results), false)
}

var (
	errorType  = types.Universe.Lookup("error").Type()
	stringType = types.Typ[types.String]
	boolType   = types.Typ[types.Bool]
)

// nativeMethods are the signatures runtimes look up by plain name, such as
// Error when printing a panic value or Unwrap in errors.Is.
var nativeMethods = map[string]*types.Signature{
	"Error":        signature(nil, stringType),
	"String":       signature(nil, stringType),
	"Unwrap":       signature(nil, errorType),
	"Is":           signature([]types.Type{errorType}, boolType),
	"RuntimeError": signature(nil),
	"Timeout":      signature(nil, boolType),
	"Temporary":    signature(nil, boolType),
}

// methodSignature is one distinct signature of a method name. spelling is
// the smallest spelling seen, so aliases and visit order do not change it.
type methodSignature struct {
	spelling string
	plain    bool
}

func bareSignature(fn *types.Func) (*types.Signature, string) {
	sig := fn.Signature()
	bare := types.NewSignatureType(nil, nil, nil, sig.Params(), sig.Results(), sig.Variadic())
	return bare, types.TypeString(bare, func(p *types.Package) string { return p.Path() })
}

// RecordMethod registers a method the program may dispatch. external marks
// methods implemented outside the compiled source, whose names runtimes use
// as is.
func (ts *Types) RecordMethod(fn *types.Func, external bool) {
	if ts.methods == nil {
		ts.methods = map[string]*typeutil.Map{}
	}
	sigs := ts.methods[fn.Name()]
	if sigs == nil {
		sigs = &typeutil.Map{}
		ts.methods[fn.Name()] = sigs
	}
	bare, spelling := bareSignature(fn)
	e, _ := sigs.At(bare).(*methodSignature)
	if e == nil {
		e = &methodSignature{spelling: spelling}
		sigs.Set(bare, e)
	} else if spelling < e.spelling {
		e.spelling = spelling
	}
	if native := nativeMethods[fn.Name()]; external || native != nil && types.Identical(bare, native) {
		e.plain = true
	}
}

// MethodID is the identity targets use to match methods in dispatch tables
// and interface checks. Targets compare identities as strings, so when a
// program uses one method name with several signatures, each signature other
// than the native or external one carries a short fingerprint.
func (ts *Types) MethodID(fn *types.Func) string {
	sigs := ts.methods[fn.Name()]
	if sigs == nil || sigs.Len() < 2 {
		return fn.Id()
	}
	bare, spelling := bareSignature(fn)
	if e, ok := sigs.At(bare).(*methodSignature); ok {
		if e.plain {
			return fn.Id()
		}
		spelling = e.spelling
	}
	h := fnv.New32a()
	h.Write([]byte(spelling))
	return fn.Id() + "#" + strconv.FormatUint(uint64(h.Sum32()), 16)
}
