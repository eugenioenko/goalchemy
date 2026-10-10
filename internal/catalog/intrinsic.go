package catalog

import "go/types"

// ErrorsPackage is the standard errors package compiled as source.
const ErrorsPackage = StdModule + "errors"

// IsErrorsAs reports whether obj is errors.As from std/errors or its native
// lib/errors counterpart. Go implements As with reflection, so the compiler
// expands each call for the static type of its target instead.
func IsErrorsAs(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Name() != "As" || fn.Pkg() == nil {
		return false
	}
	return fn.Pkg().Path() == ErrorsPackage || fn.Pkg().Path() == LibModule+"errors"
}

// JSONPackage is the standard encoding/json package compiled as source.
const JSONPackage = StdModule + "encoding/json"

// JSONMarshal reports whether obj is json.Marshal, json.MarshalIndent or
// json.Unmarshal from std/encoding/json or its native lib/json counterpart,
// returning the name.
// The compiler describes the static type of the operand instead of using
// reflection.
func JSONMarshal(obj types.Object) (string, bool) {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Signature().Recv() != nil || fn.Name() != "Marshal" && fn.Name() != "MarshalIndent" && fn.Name() != "Unmarshal" {
		return "", false
	}
	p := fn.Pkg().Path()
	return fn.Name(), p == JSONPackage || p == LibModule+"json"
}
